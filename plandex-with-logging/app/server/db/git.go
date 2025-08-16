package db

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	maxGitRetries     = 5
	baseGitRetryDelay = 100 * time.Millisecond
)

func init() {
	// ensure git is available
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("error running git --version: %v", err))
	}
}

type GitRepo struct {
	orgId  string
	planId string
}

func InitGitRepo(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)
	return initGitRepo(dir)
}

func initGitRepo(dir string) error {
	// Set the default branch name to 'main' for the new repository
	res, err := exec.Command("git", "-C", dir, "init", "-b", "main").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error initializing git repository with 'main' as default branch for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	// Configure user name and email for the repository
	if err := setGitConfig(dir, "user.email", "server@plandex.ai"); err != nil {
		return err
	}
	if err := setGitConfig(dir, "user.name", "Plandex"); err != nil {
		return err
	}

	return nil
}

func getGitRepo(orgId, planId string) *GitRepo {
	return &GitRepo{
		orgId:  orgId,
		planId: planId,
	}
}

func (repo *GitRepo) GitAddAndCommit(branch, message string) error {
	log.Printf("[Git] GitAddAndCommit - orgId: %s, planId: %s, branch: %s, message: %s", repo.orgId, repo.planId, branch, message)
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		return gitAdd(dir, ".")
	}, dir, fmt.Sprintf("GitAddAndCommit > gitAdd: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error adding files to git repository for dir: %s, err: %v", dir, err)
	}

	err = gitWriteOperation(func() error {
		return gitCommit(dir, message)
	}, dir, fmt.Sprintf("GitAddAndCommit > gitCommit: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error committing files to git repository for dir: %s, err: %v", dir, err)
	}

	// log.Println("[Git] GitAddAndCommit - finished, logging repo state")

	// repo.LogGitRepoState()

	return nil
}

func (repo *GitRepo) GitRewindToSha(branch, sha string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		return gitRewindToSha(dir, sha)
	}, dir, fmt.Sprintf("GitRewindToSha > gitRewindToSha: plan=%s branch=%s", planId, branch))
	if err != nil {
		return fmt.Errorf("error rewinding git repository for dir: %s, err: %v", dir, err)
	}

	return nil
}

func (repo *GitRepo) GetCurrentCommitSha() (sha string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting current commit SHA for dir: %s, err: %v", dir, err)
	}

	sha = strings.TrimSpace(string(output))
	return sha, nil
}

func (repo *GitRepo) GetCommitTime(branch, ref string) (time.Time, error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	// Use git show to get the commit timestamp
	cmd := exec.Command("git", "-C", dir, "show", "-s", "--format=%ct", ref)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting commit time for ref %s: %v", ref, err)
	}

	// Parse the Unix timestamp
	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing commit timestamp for ref %s: %v", ref, err)
	}

	// Convert Unix timestamp to time.Time
	commitTime := time.Unix(timestamp, 0)
	return commitTime, nil
}

func (repo *GitRepo) GitResetToSha(sha string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		cmd := exec.Command("git", "-C", dir, "reset", "--hard", sha)
		_, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error resetting git repository to SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
		}

		return nil
	}, dir, fmt.Sprintf("GitResetToSha > gitReset: plan=%s sha=%s", planId, sha))

	if err != nil {
		return fmt.Errorf("error resetting git repository to SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
	}

	return nil
}

func (repo *GitRepo) GitCheckoutSha(sha string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		cmd := exec.Command("git", "-C", dir, "checkout", sha)
		_, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error checking out git repository at SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
		}

		return nil
	}, dir, fmt.Sprintf("GitCheckoutSha > gitCheckout: plan=%s sha=%s", planId, sha))

	if err != nil {
		return fmt.Errorf("error checking out git repository at SHA for dir: %s, sha: %s, err: %v", dir, sha, err)
	}

	return nil
}

func (repo *GitRepo) GetGitCommitHistory(branch string) (body string, shas []string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	body, shas, err = getGitCommitHistory(dir, branch)
	if err != nil {
		return "", nil, fmt.Errorf("error getting git history for dir: %s, branch: %s, err: %v", dir, branch, err)
	}

	return body, shas, nil
}

func (repo *GitRepo) GetLatestCommit(branch string) (sha, body string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	sha, body, err = getLatestCommit(dir, branch)
	if err != nil {
		return "", "", fmt.Errorf("error getting latest commit for dir: %s, branch: %s, err: %v", dir, branch, err)
	}

	return sha, body, nil
}

func (repo *GitRepo) GetLatestCommitShaBeforeTime(branch string, before time.Time) (sha string, err error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	log.Printf("ADMIN - GetLatestCommitShaBeforeTime - dir: %s, before: %s", dir, before.Format("2006-01-02T15:04:05Z"))

	// Round up to the next second
	// roundedTime := before.Add(time.Second).Truncate(time.Second)

	gitFormattedTime := before.Format("2006-01-02 15:04:05+0000")

	// log.Printf("ADMIN - Git formatted time: %s", gitFormattedTime)

	cmd := exec.Command("git", "-C", dir, "log", "-n", "1",
		"--before="+gitFormattedTime,
		"--pretty=%h@@|@@%B@>>>@")
	log.Printf("ADMIN - Executing command: %s", cmd.String())
	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting latest commit before time for dir: %s, err: %v, output: %s", dir, err, string(res))
	}

	// log.Printf("ADMIN - git log res: %s", string(res))

	output := strings.TrimSpace(string(res))

	// history := processGitHistoryOutput(strings.TrimSpace(string(res)))

	// log.Printf("ADMIN - History: %v", history)

	if output == "" {
		return "", fmt.Errorf("no commits found before time: %s", before.Format("2006-01-02T15:04:05Z"))
	}

	sha = strings.Split(output, "@@|@@")[0]
	return sha, nil
}

func (repo *GitRepo) GitListBranches() ([]string, error) {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	var out bytes.Buffer
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = dir
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error getting git branches for dir: %s, err: %v", dir, err)
	}

	branches := strings.Split(strings.TrimSpace(out.String()), "\n")

	if len(branches) == 0 || (len(branches) == 1 && branches[0] == "") {
		return []string{"main"}, nil
	}

	return branches, nil
}

func (repo *GitRepo) GitCreateBranch(newBranch string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		res, err := exec.Command("git", "-C", dir, "checkout", "-b", newBranch).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error creating git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
		}

		return nil
	}, dir, fmt.Sprintf("GitCreateBranch > gitCheckout: plan=%s branch=%s", planId, newBranch))

	if err != nil {
		return err
	}

	return nil
}

func (repo *GitRepo) GitDeleteBranch(branchName string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		res, err := exec.Command("git", "-C", dir, "branch", "-D", branchName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error deleting git branch for dir: %s, err: %v, output: %s", dir, err, string(res))
		}

		return nil
	}, dir, fmt.Sprintf("GitDeleteBranch > gitBranch: plan=%s branch=%s", planId, branchName))

	if err != nil {
		return err
	}

	return nil
}

func (repo *GitRepo) GitClearUncommittedChanges(branch string) error {
	orgId := repo.orgId
	planId := repo.planId

	log.Printf("[Git] GitClearUncommittedChanges - orgId: %s, planId: %s, branch: %s", orgId, planId, branch)

	dir := getPlanDir(orgId, planId)

	// first do a lightweight git status to check if there are any uncommitted changes
	// prevents heavier operations below if there are no changes (the usual case)
	res, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error checking for uncommitted changes: %v, output: %s", err, string(res))
	}

	// If there's output, there are uncommitted changes
	hasChanges := strings.TrimSpace(string(res)) != ""

	if !hasChanges {
		log.Printf("[Git] GitClearUncommittedChanges - no changes to clear for plan %s", planId)
		return nil
	}

	err = gitWriteOperation(func() error {
		// Reset staged changes
		log.Printf("[Git] GitClearUncommittedChanges - resetting staged changes for plan %s", planId)
		res, err := exec.Command("git", "-C", dir, "reset", "--hard").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error resetting staged changes | err: %v, output: %s", err, string(res))
		}
		log.Printf("[Git] GitClearUncommittedChanges - reset staged changes finished for plan %s", planId)
		return nil
	}, dir, fmt.Sprintf("GitClearUncommittedChanges > gitReset: plan=%s", planId))

	if err != nil {
		return err
	}

	err = gitWriteOperation(func() error {
		// Clean untracked files
		log.Printf("[Git] GitClearUncommittedChanges - cleaning untracked files for plan %s", planId)
		res, err := exec.Command("git", "-C", dir, "clean", "-d", "-f").CombinedOutput()
		if err != nil {
			return fmt.Errorf("error cleaning untracked files | err: %v, output: %s", err, string(res))
		}
		log.Printf("[Git] GitClearUncommittedChanges - clean untracked files finished for plan %s", planId)
		return nil
	}, dir, fmt.Sprintf("GitClearUncommittedChanges > gitClean: plan=%s", planId))

	return err
}

func (repo *GitRepo) GitCheckoutBranch(branch string) error {
	orgId := repo.orgId
	planId := repo.planId

	dir := getPlanDir(orgId, planId)

	err := gitWriteOperation(func() error {
		return gitCheckoutBranch(dir, branch)
	}, dir, fmt.Sprintf("GitCheckoutBranch > gitCheckout: plan=%s branch=%s", planId, branch))

	if err != nil {
		return err
	}

	return nil
}

func gitAdd(repoDir, path string) error {
	cmd := exec.Command("git", "add", path)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding %s to git: %v, output: %s", path, err, string(output))
	}
	return nil
}

func gitCommit(repoDir, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error committing changes: %v, output: %s", err, string(output))
	}
	return nil
}

func gitRewindToSha(repoDir, sha string) error {
	cmd := exec.Command("git", "reset", "--hard", sha)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error rewinding to sha %s: %v, output: %s", sha, err, string(output))
	}
	return nil
}

func getGitCommitHistory(repoDir, branch string) (string, []string, error) {
	// Get commit messages
	cmd := exec.Command("git", "log", branch, "--pretty=format:%h %s", "-n", "10")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("error getting commit history: %v, output: %s", err, string(output))
	}

	// Get commit SHAs
	cmd = exec.Command("git", "log", branch, "--pretty=format:%H", "-n", "10")
	cmd.Dir = repoDir
	shaOutput, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("error getting commit SHAs: %v, output: %s", err, string(shaOutput))
	}

	shas := strings.Split(strings.TrimSpace(string(shaOutput)), "\n")
	return string(output), shas, nil
}

func getLatestCommit(repoDir, branch string) (string, string, error) {
	// Get commit SHA
	shaCmd := exec.Command("git", "rev-parse", branch)
	shaCmd.Dir = repoDir
	shaOutput, err := shaCmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("error getting latest commit SHA: %v, output: %s", err, string(shaOutput))
	}
	sha := strings.TrimSpace(string(shaOutput))

	// Get commit message
	msgCmd := exec.Command("git", "show", "-s", "--format=%B", sha)
	msgCmd.Dir = repoDir
	msgOutput, err := msgCmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("error getting latest commit message: %v, output: %s", err, string(msgOutput))
	}

	return sha, strings.TrimSpace(string(msgOutput)), nil
}

func gitCheckoutBranch(repoDir, branch string) error {
	log.Printf("[Git] gitCheckoutBranch - repoDir: %s, branch: %s", repoDir, branch)
	
	// Create the directory if it doesn't exist
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		log.Printf("[Git] Creating repository directory: %s", repoDir)
		err := os.MkdirAll(repoDir, 0755)
		if err != nil {
			err = fmt.Errorf("failed to create repository directory %s: %v", repoDir, err)
			log.Printf("[Git] gitCheckoutBranch error: %v", err)
			return err
		}
	}

	// Check if the repository is valid
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Initialize a new git repository if it doesn't exist
		log.Printf("[Git] Initializing new git repository in %s", repoDir)
		if err := initGitRepo(repoDir); err != nil {
			err = fmt.Errorf("error initializing git repository: %v", err)
			log.Printf("[Git] gitCheckoutBranch error: %v", err)
			return err
		}
	}

	// Get current branch for debugging
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	currentBranch, _ := cmd.Output()
	log.Printf("[Git] Current branch before checkout: %s", strings.TrimSpace(string(currentBranch)))

	// Check if the branch exists
	cmd = exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoDir
	err := cmd.Run()

	if err != nil {
		log.Printf("[Git] Branch %s does not exist, creating new branch", branch)
		// Branch doesn't exist, create it
		cmd = exec.Command("git", "checkout", "-b", branch)
	} else {
		// Branch exists, check it out
		log.Printf("[Git] Branch %s exists, checking it out", branch)
		cmd = exec.Command("git", "checkout", branch)
	}

	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error checking out branch %s: %v, output: %s", branch, err, string(output))
		log.Printf("[Git] gitCheckoutBranch error: %v", err)
		
		// Additional debug: List all branches
		cmd = exec.Command("git", "branch", "-a")
		cmd.Dir = repoDir
		branches, _ := cmd.CombinedOutput()
		log.Printf("[Git] Available branches: %s", string(branches))
		
		return err
	}

	log.Printf("[Git] Successfully checked out branch %s", branch)
	return nil
}

func removeLockFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}

	log.Printf("[Git] Removing lock file: %s", path)
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("error removing lock file %s: %v", path, err)
	}

	// Verify the file was actually removed
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("lock file %s still exists after removal", path)
	}

	log.Printf("[Git] Successfully removed lock file: %s", path)
	return nil
}

func gitRemoveIndexLockFileIfExists(repoDir string) error {
	log.Printf("[Git] gitRemoveIndexLockFileIfExists - repoDir: %s", repoDir)

	paths := []string{
		filepath.Join(repoDir, ".git", "index.lock"),
		filepath.Join(repoDir, ".git", "refs", "heads", "HEAD.lock"),
		filepath.Join(repoDir, ".git", "HEAD.lock"),
	}

	errCh := make(chan error, len(paths))

	for _, path := range paths {
		go func(path string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in gitRemoveIndexLockFileIfExists: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in gitRemoveIndexLockFileIfExists: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			if err := removeLockFile(path); err != nil {
				errCh <- err
				return
			}
			errCh <- nil
		}(path)
	}

	errs := []error{}
	for i := 0; i < len(paths); i++ {
		err := <-errCh
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("error removing lock files: %v", errs)
	}

	return nil
}

func setGitConfig(repoDir, key, value string) error {
	res, err := exec.Command("git", "-C", repoDir, "config", key, value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting git config %s to %s for dir: %s, err: %v, output: %s", key, value, repoDir, err, string(res))
	}
	return nil
}

func gitWriteOperation(operation func() error, repoDir, label string) error {
	log.Printf("[Git] gitWriteOperation - label: %s", label)
	var err error
	for attempt := 0; attempt < maxGitRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseGitRetryDelay // Exponential backoff
			time.Sleep(delay)
			log.Printf("Retry attempt %d for git operation %s (delay: %v)\n", attempt+1, label, delay)
		}

		err = operation()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if strings.Contains(err.Error(), "index.lock") || strings.Contains(err.Error(), "cannot lock ref") {
			log.Printf("Git lock file error detected for %s, will retry: %v\n", label, err)
			err = gitRemoveIndexLockFileIfExists(repoDir)
			if err != nil {
				log.Printf("error removing lock files: %v", err)
			}
			continue
		}

		// Non-retryable error
		return err
	}
	return fmt.Errorf("operation %s failed after %d attempts: %v", label, maxGitRetries, err)
}

// LogGitRepoState prints out useful debug info about the current git repository:
//   - The currently checked-out branch
//   - The last few commits
//   - The status (untracked changes, etc.)
//   - A directory listing of refs/heads
//   - A directory listing of .git/ (to spot any leftover lock files or HEAD files)
func (repo *GitRepo) LogGitRepoState() {
	repoDir := getPlanDir(repo.orgId, repo.planId)

	log.Println("[DEBUG] --- Git Repo State ---")

	// 1. Current branch
	out, err := exec.Command("git", "-C", repoDir, "branch", "--show-current").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git branch --show-current`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] Current branch: %s", string(out))
	}

	// 2. Recent commits
	out, err = exec.Command("git", "-C", repoDir, "log", "--oneline", "-5").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git log --oneline -5`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] Recent commits:\n%s", string(out))
	}

	// 3. Git status
	out, err = exec.Command("git", "-C", repoDir, "status", "--short", "--branch").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git status`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] Git status:\n%s", string(out))
	}

	// 4. Show all refs (to see if `.git/refs/heads/HEAD` exists)
	out, err = exec.Command("git", "-C", repoDir, "show-ref").CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error running `git show-ref`: %v, output: %s", err, string(out))
	} else {
		log.Printf("[DEBUG] All refs:\n%s", string(out))
	}

	// 5. Directory listing of .git/refs/heads
	headsDir := filepath.Join(repoDir, ".git", "refs", "heads")
	out, err = exec.Command("ls", "-l", headsDir).CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error listing heads dir: %s, err: %v, output: %s", headsDir, err, string(out))
	} else {
		log.Printf("[DEBUG] .git/refs/heads contents:\n%s", string(out))
	}

	// 5a. If there's actually a HEAD file in `.git/refs/heads`, cat it.
	headRefPath := filepath.Join(headsDir, "HEAD")
	if _, err := os.Stat(headRefPath); err == nil {
		// The file `.git/refs/heads/HEAD` exists, which is unusual
		log.Printf("[DEBUG] Found .git/refs/heads/HEAD. Dumping contents:")
		catOut, _ := exec.Command("cat", headRefPath).CombinedOutput()
		log.Printf("[DEBUG] .git/refs/heads/HEAD contents:\n%s", string(catOut))
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for .git/refs/heads/HEAD: %v", err)
	}

	// 6. Directory listing of .git/ in case there's HEAD.lock or index.lock
	gitDir := filepath.Join(repoDir, ".git")
	out, err = exec.Command("ls", "-l", gitDir).CombinedOutput()
	if err != nil {
		log.Printf("[DEBUG] error listing .git dir: %s, err: %v, output: %s", gitDir, err, string(out))
	} else {
		log.Printf("[DEBUG] .git/ contents:\n%s", string(out))
	}

	// 6a. If there's a .git/HEAD file, cat it
	headFilePath := filepath.Join(gitDir, "HEAD")
	if _, err := os.Stat(headFilePath); err == nil {
		log.Printf("[DEBUG] .git/HEAD file exists. Dumping contents:")
		catOut, _ := exec.Command("cat", headFilePath).CombinedOutput()
		log.Printf("[DEBUG] .git/HEAD contents:\n%s", string(catOut))
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for .git/HEAD: %v", err)
	}

	// 6b. Check for HEAD.lock or index.lock specifically
	headLockPath := filepath.Join(gitDir, "HEAD.lock")
	if _, err := os.Stat(headLockPath); err == nil {
		log.Printf("[DEBUG] HEAD.lock file exists at: %s", headLockPath)
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for HEAD.lock: %v", err)
	}

	indexLockPath := filepath.Join(gitDir, "index.lock")
	if _, err := os.Stat(indexLockPath); err == nil {
		log.Printf("[DEBUG] index.lock file exists at: %s", indexLockPath)
	} else if !os.IsNotExist(err) {
		log.Printf("[DEBUG] error checking for index.lock: %v", err)
	}

	log.Println("[DEBUG] --- End Git Repo State ---")
}
