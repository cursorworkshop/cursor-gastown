package swarm

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Integration branch errors
var (
	ErrBranchExists     = errors.New("branch already exists")
	ErrBranchNotFound   = errors.New("branch not found")
	ErrMergeConflict    = errors.New("merge conflict")
	ErrNotOnIntegration = errors.New("not on integration branch")
)

// CreateIntegrationBranch creates the integration branch for a swarm.
// The branch is created from the swarm's BaseCommit and pushed to origin.
func (m *Manager) CreateIntegrationBranch(swarmID string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	branchName := swarm.Integration

	// Check if branch already exists
	if m.branchExists(branchName) {
		return ErrBranchExists
	}

	// Create branch from BaseCommit
	if err := m.gitRun("checkout", "-b", branchName, swarm.BaseCommit); err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	// Push to origin (non-fatal: may not have remote)
	_ = m.gitRun("push", "-u", "origin", branchName)

	return nil
}

// MergeToIntegration merges a worker branch into the integration branch.
// Returns ErrMergeConflict if the merge has conflicts.
func (m *Manager) MergeToIntegration(swarmID, workerBranch string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	// Ensure we're on the integration branch
	currentBranch, err := m.getCurrentBranch()
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}
	if currentBranch != swarm.Integration {
		if err := m.gitRun("checkout", swarm.Integration); err != nil {
			return fmt.Errorf("checking out integration: %w", err)
		}
	}

	// Fetch the worker branch (non-fatal: may not exist on remote, try local)
	_ = m.gitRun("fetch", "origin", workerBranch)

	// Attempt merge
	err = m.gitRun("merge", "--no-ff", "-m",
		fmt.Sprintf("Merge %s into %s", workerBranch, swarm.Integration),
		workerBranch)
	if err != nil {
		// Check if it's a merge conflict
		if strings.Contains(err.Error(), "CONFLICT") ||
			strings.Contains(err.Error(), "Merge conflict") {
			return ErrMergeConflict
		}
		return fmt.Errorf("merging: %w", err)
	}

	return nil
}

// AbortMerge aborts an in-progress merge.
func (m *Manager) AbortMerge() error {
	return m.gitRun("merge", "--abort")
}

// LandToMain merges the integration branch to the target branch (usually main).
func (m *Manager) LandToMain(swarmID string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	// Checkout target branch
	if err := m.gitRun("checkout", swarm.TargetBranch); err != nil {
		return fmt.Errorf("checking out %s: %w", swarm.TargetBranch, err)
	}

	// Pull latest (non-fatal: may fail if remote unreachable)
	_ = m.gitRun("pull", "origin", swarm.TargetBranch)

	// Merge integration branch
	err = m.gitRun("merge", "--no-ff", "-m",
		fmt.Sprintf("Land swarm %s", swarmID),
		swarm.Integration)
	if err != nil {
		if strings.Contains(err.Error(), "CONFLICT") {
			return ErrMergeConflict
		}
		return fmt.Errorf("merging to %s: %w", swarm.TargetBranch, err)
	}

	// Push
	if err := m.gitRun("push", "origin", swarm.TargetBranch); err != nil {
		return fmt.Errorf("pushing: %w", err)
	}

	return nil
}

// CleanupBranches removes all branches associated with a swarm.
func (m *Manager) CleanupBranches(swarmID string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	var lastErr error

	// Delete integration branch locally
	if err := m.gitRun("branch", "-D", swarm.Integration); err != nil {
		lastErr = err
	}

	// Delete integration branch remotely (best-effort cleanup)
	_ = m.gitRun("push", "origin", "--delete", swarm.Integration)

	// Delete worker branches (best-effort cleanup)
	for _, task := range swarm.Tasks {
		if task.Branch != "" {
			// Local delete
			_ = m.gitRun("branch", "-D", task.Branch)
			// Remote delete
			_ = m.gitRun("push", "origin", "--delete", task.Branch)
		}
	}

	return lastErr
}

// GetIntegrationBranch returns the integration branch name for a swarm.
func (m *Manager) GetIntegrationBranch(swarmID string) (string, error) {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return "", err
	}
	return swarm.Integration, nil
}

// GetWorkerBranch generates the branch name for a worker on a task.
func (m *Manager) GetWorkerBranch(swarmID, worker, taskID string) string {
	return fmt.Sprintf("%s/%s/%s", swarmID, worker, taskID)
}

// branchExists checks if a branch exists locally.
func (m *Manager) branchExists(branch string) bool {
	err := m.gitRun("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// getCurrentBranch returns the current branch name.
func (m *Manager) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = m.gitDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// gitRun executes a git command.
func (m *Manager) gitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = m.gitDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s: %s", args[0], errMsg)
		}
		return fmt.Errorf("%s: %w", args[0], err)
	}

	return nil
}
