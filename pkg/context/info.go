package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds context information about where a query is being run
type Info struct {
	WorkingDir string
	GitRepo    string
	GitBranch  string
	GitCommit  string
}

// GetContextInfo gathers context information about the current execution environment
func GetContextInfo(workDir string) *Info {
	info := &Info{
		WorkingDir: workDir,
	}
	
	// If workDir is empty, use current directory
	if workDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			info.WorkingDir = cwd
		}
	}
	
	// Get Git information
	if info.WorkingDir != "" {
		// Get remote URL (repo)
		if output, err := runGitCommand(info.WorkingDir, "remote", "get-url", "origin"); err == nil {
			info.GitRepo = cleanGitURL(strings.TrimSpace(output))
		}
		
		// Get current branch
		if output, err := runGitCommand(info.WorkingDir, "branch", "--show-current"); err == nil {
			info.GitBranch = strings.TrimSpace(output)
		}
		
		// Get current commit hash (short)
		if output, err := runGitCommand(info.WorkingDir, "rev-parse", "--short", "HEAD"); err == nil {
			info.GitCommit = strings.TrimSpace(output)
		}
	}
	
	return info
}

// runGitCommand runs a git command in the specified directory
func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	return string(output), err
}

// cleanGitURL removes credentials and converts to a clean repo identifier
func cleanGitURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")
	
	// Handle SSH URLs (git@github.com:user/repo)
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			return host + "/" + parts[1]
		}
	}
	
	// Handle HTTPS URLs
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
		// Remove any credentials (username:password@)
		if atIndex := strings.Index(url, "@"); atIndex != -1 {
			url = url[atIndex+1:]
		}
	}
	
	return url
}

// GetCaller attempts to determine the calling application
func GetCaller() string {
	// Check if we're running from grove-flow
	if exe, err := os.Executable(); err == nil {
		baseName := filepath.Base(exe)
		if baseName == "grove" || strings.HasPrefix(baseName, "grove-") {
			return baseName
		}
		
		// Check parent process name if possible
		// This is platform-specific and would need different implementations
		
		// Default to the executable name
		return baseName
	}
	
	return "unknown"
}