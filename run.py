#!/usr/bin/env python3
"""
Python Script Launcher

This script manages a Python virtual environment and runs the specified script
with all dependencies properly installed.
"""
import json
import logging
import os
import platform
import shutil
import subprocess
import os
import sys
import time
import venv
from pathlib import Path
from typing import Optional, Union

# Set up logging first
logging.basicConfig(
    level=logging.DEBUG,  # Set to DEBUG to see all messages
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

# Set specific log levels for our application
logging.getLogger('eterneon').setLevel(logging.DEBUG)
logging.getLogger('services').setLevel(logging.DEBUG)

# Reduce verbosity of some noisy loggers
logging.getLogger('urllib3').setLevel(logging.WARNING)
logging.getLogger('PIL').setLevel(logging.WARNING)

# Add the project root and src directory to the Python path
project_root = os.path.abspath(os.path.dirname(__file__))
src_dir = os.path.join(project_root, 'src')

# Add both the project root and src directory to the path
for path in [project_root, src_dir]:
    if path not in sys.path:
        sys.path.insert(0, path)
        logger.info(f"Added to Python path: {path}")

# Set the PYTHONPATH environment variable
os.environ['PYTHONPATH'] = os.pathsep.join([project_root, src_dir] + sys.path[1:])

# Constants
VENV_NAME = ".venv"
REQUIREMENTS_FILE = "requirements.txt"


def get_venv_path() -> Path:
    """Get the path to the virtual environment."""
    # Store venv in the project directory
    return Path(__file__).parent / VENV_NAME


def is_windows() -> bool:
    """Check if running on Windows."""
    return platform.system() == "Windows"


def get_python_executable(venv_path: Path) -> Path:
    """Get the path to the Python executable in the virtual environment."""
    if is_windows():
        python_exe = venv_path / "Scripts" / "python.exe"
        if not python_exe.exists():
            raise FileNotFoundError(f"Python executable not found at {python_exe}")
        return python_exe
    return venv_path / "bin" / "python"


def get_pip_command(venv_path: Path) -> list[str]:
    """
    Get the correct command to run pip using the virtual environment's Python.
    Returns a list of command parts suitable for subprocess.
    """
    python_exec = get_python_executable(venv_path)
    return [str(python_exec), "-m", "pip"]


def create_venv() -> Path:
    """
    Create or reuse virtual environment and install dependencies.
    
    Returns:
        Path to the created virtual environment
    """
    venv_path = get_venv_path()
    
    if not venv_path.exists():
        logger.info("=" * 60)
        logger.info("Setting up Python virtual environment...")
        logger.info(f"Creating virtual environment at: {venv_path}")
        
        # Clean up any partially created environment
        if venv_path.exists():
            shutil.rmtree(venv_path)
        
        try:
            # Use standard venv for better Windows compatibility
            logger.info("Creating virtual environment using Python's built-in venv...")
            subprocess.run(
                [sys.executable, "-m", "venv", str(venv_path)],
                check=True,
                capture_output=True,
                text=True
            )
            logger.info("✓ Virtual environment created successfully")
            
            # Install requirements
            requirements_path = Path(REQUIREMENTS_FILE)
            if requirements_path.exists():
                logger.info("\nInstalling project dependencies...")
                logger.info("This may take a few minutes for the first run...")
                
                # Get the pip command using the virtual environment's Python
                pip_cmd = get_pip_command(venv_path)
                logger.info(f"Using Python at: {pip_cmd[0]}")
                
                try:
                    # Check pip version first
                    logger.info("Checking pip installation...")
                    version_result = subprocess.run(
                        [*pip_cmd, "--version"],
                        check=True,
                        capture_output=True,
                        text=True
                    )
                    logger.info(f"Using {version_result.stdout.strip()}")
                    
                    # Upgrade pip if needed
                    logger.info("Ensuring pip is up to date...")
                    subprocess.run(
                        [*pip_cmd, "install", "--upgrade", "pip"],
                        check=True,
                        capture_output=True,
                        text=True
                    )
                    
                    # Install requirements
                    logger.info("Installing project dependencies...")
                    install_cmd = [*pip_cmd, "install", "-r", str(requirements_path)]
                    result = subprocess.run(
                        install_cmd,
                        check=True,
                        capture_output=True,
                        text=True
                    )
                    logger.debug(f"pip install output: {result.stdout}")
                    
                except subprocess.CalledProcessError as e:
                    logger.error("\n" + "!" * 60)
                    logger.error("Failed to install dependencies:")
                    logger.error(f"Command: {' '.join(e.cmd) if hasattr(e, 'cmd') else 'Unknown'}")
                    if e.stdout:
                        logger.error(f"Output: {e.stdout}")
                    if e.stderr:
                        logger.error(f"Error: {e.stderr}")
                    logger.error("!" * 60)
                    raise
                logger.info("✓ Dependencies installed successfully!")
                logger.info("=" * 60 + "\n")
            
            return venv_path
            
        except subprocess.CalledProcessError as e:
            logger.error("\n" + "!" * 60)
            logger.error("Failed to set up the virtual environment:")
            logger.error(f"Command failed: {e.cmd}")
            if e.stdout:
                logger.error(f"Output: {e.stdout}")
            if e.stderr:
                logger.error(f"Error: {e.stderr}")
            if venv_path.exists():
                shutil.rmtree(venv_path)
            logger.error("!" * 60)
            sys.exit(1)
            
        except Exception as e:
            logger.error("\n" + "!" * 60)
            logger.error("An unexpected error occurred:")
            logger.error(str(e))
            if venv_path.exists():
                shutil.rmtree(venv_path)
            logger.error("!" * 60)
            sys.exit(1)
    
    else:
        logger.info(f"Using existing virtual environment at {venv_path}")
    
    return venv_path


def check_requirements_installed(venv_path: Path, requirements_path: Path) -> bool:
    """Check if all requirements are already installed."""
    start_time = time.time()
    
    try:
        pip_cmd = get_pip_command(venv_path)
        # Get installed packages (single subprocess call)
        result = subprocess.run(
            [*pip_cmd, "list", "--format=json"],
            capture_output=True,
            text=True,
            check=True
        )
        
        # Parse JSON output for faster processing
        installed_packages = {
            pkg['name'].lower(): pkg['version'] 
            for pkg in json.loads(result.stdout)
        }
        
        # Read requirements file once
        with open(requirements_path, 'r') as f:
            requirements = [
                line.strip() 
                for line in f 
                if line.strip() and not line.strip().startswith('#')
            ]
        
        # Check all requirements
        for line in requirements:
            # Extract package name (handle cases like package[extra]>=1.0.0)
            pkg_name = line.split('[')[0].split('>=')[0].split('==')[0].split('<=')[0].strip()
            if pkg_name.lower() not in installed_packages:
                logger.debug(f"Package not found: {pkg_name}")
                return False
        
        # Only run pip check if we're in debug mode (it's slow)
        if logger.level <= logging.DEBUG:
            result = subprocess.run(
                [*pip_cmd, "check"],
                capture_output=True,
                text=True
            )
            if result.returncode != 0:
                logger.debug(f"Dependency check warnings: {result.stderr}")
        
        logger.debug(f"Requirements check completed in {time.time() - start_time:.2f}s")
        return True
        
    except subprocess.CalledProcessError as e:
        logger.debug(f"Error checking packages: {e.stderr}")
        return False
    except json.JSONDecodeError as e:
        logger.debug(f"Error parsing pip list output: {e}")
        return False
    except Exception as e:
        logger.debug(f"Unexpected error: {str(e)}")
        return False

def install_requirements(venv_path: Path, background: bool = False) -> Union[bool, subprocess.Popen, None]:
    """
    Install requirements from requirements.txt using pip.
    
    Args:
        venv_path: Path to the virtual environment
        background: If True, returns immediately with the Popen object
        
    Returns:
        If background=True: Returns Popen object or None on failure
        If background=False: Returns bool indicating success
    """
    requirements_path = Path(REQUIREMENTS_FILE)
    if not requirements_path.exists():
        logger.error(f"No {REQUIREMENTS_FILE} found in {os.getcwd()}")
        return None if background else False

    pip_cmd = get_pip_command(venv_path)
    
    # Skip check if running in background
    if not background and check_requirements_installed(venv_path, requirements_path):
        logger.info("All requirements are already installed and up to date")
        return None if background else True
    
    try:
        logger.info("Installing requirements using pip...")
        cmd = [*pip_cmd, "install", "-r", str(requirements_path)]
        
        if background:
            # Run in background and return the process
            return subprocess.Popen(
                cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
        else:
            # Run synchronously
            result = subprocess.run(
                cmd,
                check=True,
                capture_output=True,
                text=True
            )
            logger.info("Requirements installed/verified successfully")
            logger.debug(result.stdout)
            return True
            
    except subprocess.CalledProcessError as e:
        logger.error("\n" + "!" * 60)
        logger.error("Failed to install requirements:")
        logger.error(f"Command: {' '.join(e.cmd) if hasattr(e, 'cmd') else 'Unknown'}")
        if e.stdout:
            logger.error(f"Output: {e.stdout}")
        if e.stderr:
            logger.error(f"Error: {e.stderr}")
        logger.error("!" * 60)
        return None if background else False
    except Exception as e:
        logger.error(f"Unexpected error installing requirements: {str(e)}")
        return None if background else False


def run_script(venv_path: Path, script_path: Path, args: list) -> None:
    """Run a Python script in the virtual environment and open browser."""
    python_exec = get_python_executable(venv_path)
    
    if not python_exec.exists():
        logger.error(f"Python executable not found at {python_exec}")
        logger.error("Please ensure the virtual environment is properly set up.")
        return False
    
    if not script_path.exists():
        logger.error(f"Script not found at {script_path}")
        return False
    
    # Build the command
    cmd = [str(python_exec), "-u", str(script_path)] + args
    logger.info(f"Running command: {' '.join(cmd)}")
    
    try:
        # Start the script in a new process
        process = subprocess.Popen(
            cmd,
            cwd=os.path.dirname(script_path),
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,  # Combine stdout and stderr
            text=True,
            bufsize=1,
            universal_newlines=True,
            env={
                **os.environ,
                'FLASK_APP': str(script_path),
                'FLASK_DEBUG': '1',
                'PYTHONUNBUFFERED': '1'
            }
        )
        
        # Wait a bit for the server to start
        server_started = False
        start_time = time.time()
        timeout = 10  # seconds
        
        while time.time() - start_time < timeout:
            output = process.stdout.readline()
            if output:
                print(output.strip())
                if "Running on" in output or "* Running on" in output:
                    server_started = True
                    break
            time.sleep(0.1)
        
        if not server_started:
            logger.error("Server did not start within the expected time. Check the logs above for errors.")
            process.terminate()
            return False
        
        # Open the browser
        open_browser("http://localhost:5000")
        
        # Stream the output
        try:
            while True:
                output = process.stdout.readline()
                if output == '' and process.poll() is not None:
                    break
                if output:
                    print(output.strip(), flush=True)
        except KeyboardInterrupt:
            logger.info("\nShutting down server...")
            process.terminate()
            return True
        
        # Check for errors
        _, stderr = process.communicate()
        if process.returncode != 0:
            logger.error(f"Script failed with error: {stderr}")
            return False
            
        return True
        
    except Exception as e:
        logger.error(f"Error running script: {e}", exc_info=True)
        return False


def open_browser(url: str) -> None:
    """Open the specified URL in the default browser, working in both WSL and PowerShell."""
    system = platform.system()
    is_wsl = 'microsoft' in platform.release().lower() or 'wsl' in platform.release().lower()
    
    try:
        if system == 'Windows' or is_wsl:
            # On Windows or WSL
            if is_wsl:
                # In WSL, try Windows Chrome first
                chrome_paths = [
                    "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",  # Standard Windows Chrome path in WSL
                    "/mnt/c/Program Files (x86)/Google/Chrome/Application/chrome.exe",  # 32-bit Chrome
                    "/usr/bin/google-chrome",  # Linux Chrome if installed in WSL
                    "/usr/bin/chromium-browser"  # Chromium in WSL
                ]
                
                for chrome_path in chrome_paths:
                    if os.path.exists(chrome_path):
                        try:
                            if chrome_path.endswith('.exe'):
                                # For Windows Chrome from WSL
                                subprocess.Popen([chrome_path, '--new-window', url], 
                                               stdout=subprocess.PIPE, 
                                               stderr=subprocess.PIPE)
                            else:
                                # For Linux Chrome/Chromium in WSL
                                subprocess.Popen([chrome_path, '--new-window', url],
                                               stdout=subprocess.PIPE,
                                               stderr=subprocess.PIPE)
                            logger.info(f"Opened Chrome with URL: {url}")
                            return
                        except Exception as e:
                            logger.warning(f"Failed to open Chrome at {chrome_path}: {str(e)}")
            
            # Fallback to default Windows browser in WSL or native Windows
            try:
                if is_wsl:
                    # In WSL, use wslview if available
                    subprocess.Popen(['wslview', url],
                                   stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE)
                    logger.info(f"Opened default browser with URL: {url}")
                else:
                    # Native Windows
                    import webbrowser
                    webbrowser.open(url)
                    logger.info(f"Opened default browser with URL: {url}")
                return
            except Exception as e:
                logger.warning(f"Failed to open default browser: {str(e)}")
        else:
            # On Linux or macOS
            import webbrowser
            webbrowser.open(url)
            logger.info(f"Opened default browser with URL: {url}")
            return
            
    except Exception as e:
        logger.error(f"Error opening browser: {str(e)}")
    
    # If all else fails, just print the URL
    logger.warning("Could not open browser automatically. Please open manually:")
    print(f"\n  {url}\n")


def run_script(venv_path: Path, script_path: Path, args: list) -> None:
    """Run a Python script in the virtual environment and open browser."""
    import signal
    import threading
    import time
    
    python_exec = get_python_executable(venv_path)
    process = None
    shutdown_event = threading.Event()
    
    def signal_handler(sig, frame):
        """Handle termination signals to ensure clean shutdown."""
        nonlocal process
        print("\nShutting down server...")
        
        if process and process.poll() is None:
            # Send shutdown request to the server
            try:
                import requests
                requests.post('http://localhost:5000/shutdown', timeout=2)
            except Exception as e:
                print(f"Error sending shutdown request: {e}")
            
            # Give it a moment to shut down gracefully
            time.sleep(1)
            
            # Then terminate the process
            try:
                process.terminate()
                process.wait(timeout=2)
            except (subprocess.TimeoutExpired, ProcessLookupError):
                try:
                    process.kill()
                except:
                    pass
        
        sys.exit(0)
    
    # Register signal handlers for clean shutdown
    import signal
    signal.signal(signal.SIGINT, signal_handler)   # Ctrl+C
    signal.signal(signal.SIGTERM, signal_handler)  # kill command
    
    if platform.system() == 'Windows':
        signal.signal(signal.SIGBREAK, signal_handler)
    
    # Build the command
    cmd = [str(python_exec), str(script_path)]
    
    # Open the browser after a short delay to allow the server to start
    browser_opened = False
    
    def open_browser_delayed():
        nonlocal browser_opened
        if browser_opened:
            return
            
        import time
        max_attempts = 10
        for _ in range(max_attempts):
            try:
                # Try to connect to the server
                import socket
                s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                s.settimeout(1)
                result = s.connect_ex(('127.0.0.1', 5000))
                s.close()
                if result == 0:  # Port is open
                    if not browser_opened:  # Double-check to be extra safe
                        browser_opened = True
                        open_browser("http://localhost:5000/")
                    return
            except Exception as e:
                pass
            time.sleep(0.5)  # Wait before retry
            
        if not browser_opened:
            print("\nWarning: Could not connect to the server. Please check if the server started correctly.")
    
    # Start the browser in a separate thread after the server is running
    browser_thread = threading.Thread(target=open_browser_delayed)
    browser_thread.daemon = True
    browser_thread.start()
    
    process = None
    try:
        # Run the script
        process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
            universal_newlines=True,
            creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name == 'nt' else 0
        )
        
        # Forward output to console in real-time
        def forward_output(stream, prefix=''):
            try:
                for line in iter(stream.readline, ''):
                    if line:
                        print(f"{prefix}{line.rstrip()}")
            except (ValueError, IOError):
                # Stream was probably closed
                pass
        
        # Start output forwarding threads
        stdout_thread = threading.Thread(
            target=forward_output,
            args=(process.stdout, '')
        )
        stderr_thread = threading.Thread(
            target=forward_output,
            args=(process.stderr, 'ERROR: ')
        )
        
        stdout_thread.daemon = True
        stderr_thread.daemon = True
        stdout_thread.start()
        stderr_thread.start()
        
        try:
            # Wait for process to complete
            while process.poll() is None and not shutdown_event.is_set():
                time.sleep(0.1)
        except KeyboardInterrupt:
            print("\nReceived keyboard interrupt. Shutting down...")
            signal_handler(signal.SIGINT, None)
        
    except Exception as e:
        logger.error(f"Error running script: {e}")
        if process and process.poll() is None:
            try:
                process.terminate()
                process.wait(timeout=2)
            except (subprocess.TimeoutExpired, ProcessLookupError):
                try:
                    process.kill()
                except:
                    pass
    finally:
        if process and process.poll() is None:
            try:
                process.terminate()
                process.wait(timeout=2)
            except (subprocess.TimeoutExpired, ProcessLookupError):
                try:
                    process.kill()
                except:
                    pass


def list_available_scripts() -> None:
    """List available scripts in the scripts directory."""
    # TODO: Make script directory configurable instead of hardcoded 'scripts/'
    # Consider allowing scripts from root directory or other locations
    scripts_dir = Path(__file__).parent / "scripts"
    print("\nAvailable scripts in the 'scripts' directory:")
    print("-" * 50)
    
    if scripts_dir.exists():
        for script in sorted(scripts_dir.glob("*.py")):
            if script.name != "__init__.py":
                print(f"- {script.name}")
    else:
        print("No scripts directory found")
    print()


def install_system_dependencies() -> bool:
    """Install system dependencies required for Playwright in WSL."""
    if not is_windows():
        logger.info("Skipping system dependencies installation - not running in WSL")
        return True
        
    logger.info("Checking for system dependencies...")
    
    # Check if we're in WSL
    try:
        with open("/proc/version", "r") as f:
            if "microsoft" not in f.read().lower():
                logger.info("Not running in WSL, skipping system dependencies")
                return True
    except FileNotFoundError:
        logger.info("Not running in WSL, skipping system dependencies")
        return True
    
    logger.info("Installing system dependencies for Playwright in WSL...")
    
    # Define the dependencies
    dependencies = [
        "libgtk-4-1", "libgraphene-1.0-0", "libwoff2dec1.0.4", "libvpx9", "libopus0",
        "gstreamer1.0-plugins-base", "gstreamer1.0-plugins-good", "gstreamer1.0-plugins-ugly",
        "gstreamer1.0-libav", "gstreamer1.0-tools", "libgstreamer1.0-dev",
        "libgstreamer-plugins-base1.0-dev", "libflite1", "libflite1-dev",
        "libwebpdemux2", "libavif16", "libharfbuzz-icu0", "libwebpmux3",
        "libenchant-2-2", "libsecret-1-0", "libhyphen0", "libmanette-0.2-0",
        "libgles2-mesa", "libx264-164"
    ]
    
    # Install dependencies
    try:
        subprocess.run(
            ["sudo", "apt-get", "update"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        install_cmd = ["sudo", "apt-get", "install", "-y"] + dependencies
        result = subprocess.run(
            install_cmd,
            check=False,  # Don't raise exception on non-zero exit
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        if result.returncode != 0:
            logger.warning("Failed to install some system dependencies. Some Playwright features may not work.")
            logger.debug(f"Installation error: {result.stderr}")
        
        return True
        
    except (subprocess.SubprocessError, FileNotFoundError) as e:
        logger.error(f"Error installing system dependencies: {e}")
        return False

def is_uv_available() -> bool:
    """Check if uv is available in the system."""
    try:
        subprocess.run(
            ["uv", "--version"],
            capture_output=True,
            text=True,
            check=True
        )
        return True
    except (subprocess.SubprocessError, FileNotFoundError):
        return False

def install_uv() -> bool:
    """Install uv if not already installed."""
    if is_uv_available():
        return True
        
    logger.info("Installing uv for faster dependency management...")
    try:
        subprocess.run(
            ["curl", "-sSf", "https://astral.sh/uv/install.sh", "|", "sh"],
            shell=True,
            check=True
        )
        return True
    except subprocess.CalledProcessError as e:
        logger.warning(f"Failed to install uv: {e}")
        return False

def install_dependencies(venv_path: Path) -> bool:
    """Install project dependencies and test requirements using uv if available."""
    # Install system dependencies first if in WSL
    if not install_system_dependencies():
        logger.warning("Failed to install some system dependencies. Some features may not work correctly.")
    
    # Use uv if available, fall back to pip
    use_uv = is_uv_available()
    if not use_uv:
        logger.info("uv not found, falling back to pip")
    
    python_exec = get_python_executable(venv_path)
    requirements_path = Path(REQUIREMENTS_FILE)
    
    # Install base requirements
    if requirements_path.exists():
        if not check_requirements_installed(venv_path, requirements_path) or use_uv:
            logger.info("Installing requirements...")
            if use_uv:
                cmd = ["uv", "pip", "install", "-r", str(requirements_path)]
            else:
                cmd = [str(python_exec), "-m", "pip", "install", "-r", str(requirements_path)]
            
            try:
                subprocess.check_call(cmd)
            except subprocess.CalledProcessError as e:
                logger.error(f"Failed to install requirements: {e}")
                return False
    
    # Install test requirements
    test_requirements = [
        # Testing framework
        "pytest>=8.0.0",
        "pytest-cov>=4.0.0",
        "pytest-asyncio>=0.21.0",
        "pytest-mock>=3.10.0",
        "pytest-playwright>=0.2.0",
        
        # Web and API testing
        "playwright>=1.42.0",
        "requests>=2.31.0",
        "urllib3>=2.0.0",
        
        # Web framework
        "flask>=2.0.0",
        "Werkzeug>=2.0.0",
        "Jinja2>=3.0.0",
        "itsdangerous>=2.0.0",
        "click>=8.0.0",
        "markupsafe>=2.0.0",
        
        # Async support
        "asyncio>=3.4.3",
        "aiohttp>=3.9.0",
        
        # Data handling
        "python-dotenv>=1.0.0"
    ]
    
    logger.info("Installing test requirements...")
    for req in test_requirements:
        try:
            if use_uv:
                subprocess.check_call(["uv", "pip", "install", req])
            else:
                subprocess.check_call([str(python_exec), "-m", "pip", "install", req])
        except subprocess.CalledProcessError as e:
            logger.error(f"Failed to install {req}: {e}")
            return False
    
    # Install Playwright browsers
    try:
        playwright_cmd = [str(python_exec), "-m", "playwright", "install"]
        subprocess.check_call(playwright_cmd)
    except subprocess.CalledProcessError as e:
        logger.error(f"Failed to install Playwright browsers: {e}")
        return False
    
    return True


def main():
    """Main entry point with performance optimizations."""
    start_time = time.time()
    
    try:
        # Check if src/app.py exists first (fast check)
        script_path = Path("src/app.py")
        if not script_path.exists():
            logger.error("src/app.py not found in the project directory")
            sys.exit(1)
            
        # Create or reuse virtual environment
        venv_path = create_venv()
        
        # Handle --install-deps flag
        if "--install-deps" in sys.argv:
            logger.info("Installing dependencies...")
            if install_dependencies(venv_path):
                logger.info("Dependencies installed successfully")
                sys.exit(0)
            else:
                logger.error("Failed to install dependencies")
                sys.exit(1)
        
        # Check if we need to install requirements
        requirements_path = Path(REQUIREMENTS_FILE)
        if requirements_path.exists():
            if check_requirements_installed(venv_path, requirements_path):
                logger.debug("All requirements are already installed")
            else:
                logger.info("Installing requirements...")
                if not install_requirements(venv_path, background=False):
                    logger.error("Failed to install requirements")
                    sys.exit(1)
        
        # Check if a script was provided as an argument
        if len(sys.argv) > 1:
            script_path = Path(sys.argv[1])
            if script_path.exists():
                logger.info(f"Starting script: {script_path}")
                logger.info("-" * 60)
                run_script(venv_path, script_path, [arg for arg in sys.argv[2:] if arg != "--install-deps"])
            else:
                logger.error(f"Error: Script '{script_path}' not found.")
                sys.exit(1)
        else:
            # Default behavior if no script is provided
            logger.info("No script specified. Available scripts:")
            list_available_scripts()
            sys.exit(1)
            
    except KeyboardInterrupt:
        logger.info("\nOperation cancelled by user.")
        sys.exit(1)
    except Exception as e:
        logger.error("\n" + "!" * 60)
        logger.error("An unexpected error occurred:")
        logger.error(str(e))
        logger.error("!" * 60)
        logger.info("\nIf you continue to experience issues, please try:")
        logger.info("1. Deleting the .venv directory")
        logger.info("2. Running the script again")
        logger.info("3. Checking your internet connection")
        logger.info("4. Reviewing the error message above")
        sys.exit(1)


if __name__ == "__main__":
    main()
