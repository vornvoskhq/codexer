#[cfg(unix)]
use std::os::unix::process::ExitStatusExt;

use std::collections::HashMap;
use std::io;
use std::path::Path;
use std::path::PathBuf;
use std::process::ExitStatus;
use std::sync::Arc;
use std::time::Duration;
use std::time::Instant;

use async_channel::Sender;
use tokio::io::AsyncRead;
use tokio::io::AsyncReadExt;
use tokio::io::BufReader;
use tokio::process::Child;
use tokio::sync::Notify;

use crate::error::CodexErr;
use crate::error::Result;
use crate::error::SandboxErr;
use crate::protocol::Event;
use crate::protocol::EventMsg;
use crate::protocol::ExecCommandOutputDeltaEvent;
use crate::protocol::ExecOutputStream;
use crate::protocol::SandboxPolicy;
use crate::seatbelt::spawn_command_under_seatbelt;
use crate::spawn::StdioPolicy;
use crate::spawn::spawn_child_async;
use serde_bytes::ByteBuf;

// Maximum we send for each stream, which is either:
// - 10KiB OR
// - 256 lines
const MAX_STREAM_OUTPUT: usize = 10 * 1024;
const MAX_STREAM_OUTPUT_LINES: usize = 256;

const DEFAULT_TIMEOUT_MS: u64 = 10_000;

// Hardcode these since it does not seem worth including the libc crate just
// for these.
const SIGKILL_CODE: i32 = 9;
const TIMEOUT_CODE: i32 = 64;

#[derive(Debug, Clone)]
pub struct ExecParams {
    pub command: Vec<String>,
    pub cwd: PathBuf,
    pub timeout_ms: Option<u64>,
    pub env: HashMap<String, String>,
    pub with_escalated_permissions: Option<bool>,
    pub justification: Option<String>,
}

impl ExecParams {
    pub fn timeout_duration(&self) -> Duration {
        Duration::from_millis(self.timeout_ms.unwrap_or(DEFAULT_TIMEOUT_MS))
    }
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SandboxType {
    None,

    /// Only available on macOS.
    MacosSeatbelt,

    /// Only available on Linux.
    LinuxSeccomp,
}

#[derive(Clone)]
pub struct StdoutStream {
    pub sub_id: String,
    pub call_id: String,
    pub tx_event: Sender<Event>,
}

pub async fn process_exec_tool_call(
    params: ExecParams,
    sandbox_type: SandboxType,
    ctrl_c: Arc<Notify>,
    sandbox_policy: &SandboxPolicy,
    codex_linux_sandbox_exe: &Option<PathBuf>,
    stdout_stream: Option<StdoutStream>,
) -> Result<ExecToolCallOutput> {
    let start = Instant::now();

    let raw_output_result: std::result::Result<RawExecToolCallOutput, CodexErr> = match sandbox_type
    {
        SandboxType::None => exec(params, sandbox_policy, ctrl_c, stdout_stream.clone()).await,
        SandboxType::MacosSeatbelt => {
            let timeout = params.timeout_duration();
            let ExecParams {
                command, cwd, env, ..
            } = params;
            let child = spawn_command_under_seatbelt(
                command,
                sandbox_policy,
                cwd,
                StdioPolicy::RedirectForShellTool,
                env,
            )
            .await?;
            consume_truncated_output(child, ctrl_c, timeout, stdout_stream.clone()).await
        }
        SandboxType::LinuxSeccomp => {
            let timeout = params.timeout_duration();
            let ExecParams {
                command, cwd, env, ..
            } = params;

            let codex_linux_sandbox_exe = codex_linux_sandbox_exe
                .as_ref()
                .ok_or(CodexErr::LandlockSandboxExecutableNotProvided)?;
            let child = spawn_command_under_linux_sandbox(
                codex_linux_sandbox_exe,
                command,
                sandbox_policy,
                cwd,
                StdioPolicy::RedirectForShellTool,
                env,
            )
            .await?;

            consume_truncated_output(child, ctrl_c, timeout, stdout_stream).await
        }
    };
    let duration = start.elapsed();
    match raw_output_result {
        Ok(raw_output) => {
            let stdout = String::from_utf8_lossy(&raw_output.stdout).to_string();
            let stderr = String::from_utf8_lossy(&raw_output.stderr).to_string();

            #[cfg(target_family = "unix")]
            match raw_output.exit_status.signal() {
                Some(TIMEOUT_CODE) => return Err(CodexErr::Sandbox(SandboxErr::Timeout)),
                Some(signal) => {
                    return Err(CodexErr::Sandbox(SandboxErr::Signal(signal)));
                }
                None => {}
            }

            let exit_code = raw_output.exit_status.code().unwrap_or(-1);

            if exit_code != 0 && is_likely_sandbox_denied(sandbox_type, exit_code) {
                return Err(CodexErr::Sandbox(SandboxErr::Denied(
                    exit_code, stdout, stderr,
                )));
            }

            Ok(ExecToolCallOutput {
                exit_code,
                stdout,
                stderr,
                duration,
            })
        }
        Err(err) => {
            tracing::error!("exec error: {err}");
            Err(err)
        }
    }
}

/// Spawn a shell tool command under the Linux Landlock+seccomp sandbox helper
/// (codex-linux-sandbox).
///
/// Unlike macOS Seatbelt where we directly embed the policy text, the Linux
/// helper accepts a list of `--sandbox-permission`/`-s` flags mirroring the
/// public CLI. We convert the internal [`SandboxPolicy`] representation into
/// the equivalent CLI options.
pub async fn spawn_command_under_linux_sandbox<P>(
    codex_linux_sandbox_exe: P,
    command: Vec<String>,
    sandbox_policy: &SandboxPolicy,
    cwd: PathBuf,
    stdio_policy: StdioPolicy,
    env: HashMap<String, String>,
) -> std::io::Result<Child>
where
    P: AsRef<Path>,
{
    let args = create_linux_sandbox_command_args(command, sandbox_policy, &cwd);
    let arg0 = Some("codex-linux-sandbox");
    spawn_child_async(
        codex_linux_sandbox_exe.as_ref().to_path_buf(),
        args,
        arg0,
        cwd,
        sandbox_policy,
        stdio_policy,
        env,
    )
    .await
}

/// Converts the sandbox policy into the CLI invocation for `codex-linux-sandbox`.
fn create_linux_sandbox_command_args(
    command: Vec<String>,
    sandbox_policy: &SandboxPolicy,
    cwd: &Path,
) -> Vec<String> {
    #[expect(clippy::expect_used)]
    let sandbox_policy_cwd = cwd.to_str().expect("cwd must be valid UTF-8").to_string();

    #[expect(clippy::expect_used)]
    let sandbox_policy_json =
        serde_json::to_string(sandbox_policy).expect("Failed to serialize SandboxPolicy to JSON");

    let mut linux_cmd: Vec<String> = vec![
        sandbox_policy_cwd,
        sandbox_policy_json,
        // Separator so that command arguments starting with `-` are not parsed as
        // options of the helper itself.
        "--".to_string(),
    ];

    // Append the original tool command.
    linux_cmd.extend(command);

    linux_cmd
}

/// We don't have a fully deterministic way to tell if our command failed
/// because of the sandbox - a command in the user's zshrc file might hit an
/// error, but the command itself might fail or succeed for other reasons.
/// For now, we conservatively check for 'command not found' (exit code 127),
/// and can add additional cases as necessary.
fn is_likely_sandbox_denied(sandbox_type: SandboxType, exit_code: i32) -> bool {
    if sandbox_type == SandboxType::None {
        return false;
    }

    // Quick rejects: well-known non-sandbox shell exit codes
    // 127: command not found, 2: misuse of shell builtins
    if exit_code == 127 {
        return false;
    }

    // For all other cases, we assume the sandbox is the cause
    true
}

#[derive(Debug)]
pub struct RawExecToolCallOutput {
    pub exit_status: ExitStatus,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
}

#[derive(Debug)]
pub struct ExecToolCallOutput {
    pub exit_code: i32,
    pub stdout: String,
    pub stderr: String,
    pub duration: Duration,
}

async fn exec(
    params: ExecParams,
    sandbox_policy: &SandboxPolicy,
    ctrl_c: Arc<Notify>,
    stdout_stream: Option<StdoutStream>,
) -> Result<RawExecToolCallOutput> {
    let timeout = params.timeout_duration();
    let ExecParams {
        command, cwd, env, ..
    } = params;

    let (program, args) = command.split_first().ok_or_else(|| {
        CodexErr::Io(io::Error::new(
            io::ErrorKind::InvalidInput,
            "command args are empty",
        ))
    })?;
    let arg0 = None;
    let child = spawn_child_async(
        PathBuf::from(program),
        args.into(),
        arg0,
        cwd,
        sandbox_policy,
        StdioPolicy::RedirectForShellTool,
        env,
    )
    .await?;
    consume_truncated_output(child, ctrl_c, timeout, stdout_stream).await
}

/// Consumes the output of a child process, truncating it so it is suitable for
/// use as the output of a `shell` tool call. Also enforces specified timeout.
pub(crate) async fn consume_truncated_output(
    mut child: Child,
    ctrl_c: Arc<Notify>,
    timeout: Duration,
    stdout_stream: Option<StdoutStream>,
) -> Result<RawExecToolCallOutput> {
    // Both stdout and stderr were configured with `Stdio::piped()`
    // above, therefore `take()` should normally return `Some`.  If it doesn't
    // we treat it as an exceptional I/O error

    let stdout_reader = child.stdout.take().ok_or_else(|| {
        CodexErr::Io(io::Error::other(
            "stdout pipe was unexpectedly not available",
        ))
    })?;
    let stderr_reader = child.stderr.take().ok_or_else(|| {
        CodexErr::Io(io::Error::other(
            "stderr pipe was unexpectedly not available",
        ))
    })?;

    let stdout_handle = tokio::spawn(read_capped(
        BufReader::new(stdout_reader),
        MAX_STREAM_OUTPUT,
        MAX_STREAM_OUTPUT_LINES,
        stdout_stream.clone(),
        false,
    ));
    let stderr_handle = tokio::spawn(read_capped(
        BufReader::new(stderr_reader),
        MAX_STREAM_OUTPUT,
        MAX_STREAM_OUTPUT_LINES,
        stdout_stream.clone(),
        true,
    ));

    let interrupted = ctrl_c.notified();
    let exit_status = tokio::select! {
        result = tokio::time::timeout(timeout, child.wait()) => {
            match result {
                Ok(Ok(exit_status)) => exit_status,
                Ok(e) => e?,
                Err(_) => {
                    // timeout
                    child.start_kill()?;
                    // Debatable whether `child.wait().await` should be called here.
                    synthetic_exit_status(128 + TIMEOUT_CODE)
                }
            }
        }
        _ = interrupted => {
            child.start_kill()?;
            synthetic_exit_status(128 + SIGKILL_CODE)
        }
    };

    let stdout = stdout_handle.await??;
    let stderr = stderr_handle.await??;

    Ok(RawExecToolCallOutput {
        exit_status,
        stdout,
        stderr,
    })
}

async fn read_capped<R: AsyncRead + Unpin + Send + 'static>(
    mut reader: R,
    max_output: usize,
    max_lines: usize,
    stream: Option<StdoutStream>,
    is_stderr: bool,
) -> io::Result<Vec<u8>> {
    let mut buf = Vec::with_capacity(max_output.min(8 * 1024));
    let mut tmp = [0u8; 8192];

    let mut remaining_bytes = max_output;
    let mut remaining_lines = max_lines;

    loop {
        let n = reader.read(&mut tmp).await?;
        if n == 0 {
            break;
        }

        if let Some(stream) = &stream {
            let chunk = tmp[..n].to_vec();
            let msg = EventMsg::ExecCommandOutputDelta(ExecCommandOutputDeltaEvent {
                call_id: stream.call_id.clone(),
                stream: if is_stderr {
                    ExecOutputStream::Stderr
                } else {
                    ExecOutputStream::Stdout
                },
                chunk: ByteBuf::from(chunk),
            });
            let event = Event {
                id: stream.sub_id.clone(),
                msg,
            };
            #[allow(clippy::let_unit_value)]
            let _ = stream.tx_event.send(event).await;
        }

        // Copy into the buffer only while we still have byte and line budget.
        if remaining_bytes > 0 && remaining_lines > 0 {
            let mut copy_len = 0;
            for &b in &tmp[..n] {
                if remaining_bytes == 0 || remaining_lines == 0 {
                    break;
                }
                copy_len += 1;
                remaining_bytes -= 1;
                if b == b'\n' {
                    remaining_lines -= 1;
                }
            }
            buf.extend_from_slice(&tmp[..copy_len]);
        }
        // Continue reading to EOF to avoid back-pressure, but discard once caps are hit.
    }

    Ok(buf)
}

#[cfg(unix)]
fn synthetic_exit_status(code: i32) -> ExitStatus {
    use std::os::unix::process::ExitStatusExt;
    std::process::ExitStatus::from_raw(code)
}

#[cfg(windows)]
fn synthetic_exit_status(code: i32) -> ExitStatus {
    use std::os::windows::process::ExitStatusExt;
    #[expect(clippy::unwrap_used)]
    std::process::ExitStatus::from_raw(code.try_into().unwrap())
}
