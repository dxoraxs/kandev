package planfiles

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/gitstate"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

// DefaultCommitMessage is the commit message used when the owner gives none.
const DefaultCommitMessage = "docs(plans): update plan files"

// maxCommitOutputLines bounds the git output returned with a failed commit.
const maxCommitOutputLines = 20

// Git operation error codes. They are the machine-readable codes of the HTTP
// API.
const (
	CodeRepositoryBusy     = "repository_busy"
	CodeNothingToCommit    = "nothing_to_commit"
	CodeCommitFailed       = "commit_failed"
	CodeRepositoryNotFound = "repository_not_found"
	CodeInvalidCommit      = "invalid_commit"
)

// GitError is a rejected git operation with a machine-readable code. Output
// holds the last lines of git's output for a failed commit.
type GitError struct {
	Code    string
	Message string
	Output  string
}

func (e *GitError) Error() string { return e.Message }

// GitRepoStatus lists the plan files of one local repository that have
// uncommitted changes.
type GitRepoStatus struct {
	RepositoryID   string   `json:"repository_id"`
	RepositoryName string   `json:"repository_name"`
	Files          []string `json:"files"`
}

// CommitRequest is the body of POST /plan-files/commit.
type CommitRequest struct {
	RepositoryID string `json:"repository_id"`
	Message      string `json:"message,omitempty"`
}

// CommitResult describes the commit that was created.
type CommitResult struct {
	Commit string   `json:"commit"`
	Files  []string `json:"files"`
}

// gitScope is what decides which files count as plan files of a workspace: the
// scanned directories and the index file name. A workspace without a config
// uses the defaults and has no index file.
func (s *Service) gitScope(ctx context.Context, workspaceID string) (dirs []string, indexFile string, err error) {
	cfg, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil {
		return nil, "", err
	}
	if cfg == nil {
		return DefaultDirectories(), "", nil
	}
	return cfg.Directories, cfg.IndexFile, nil
}

// planDirtyFiles returns the sorted repository-relative paths of the plan files
// and the plan index under dirs that are modified, staged, or untracked. A root
// that is not a git working tree has none.
func planDirtyFiles(ctx context.Context, root string, dirs []string, indexFile string) ([]string, error) {
	dirty, err := gitstate.Dirty(ctx, root, dirs)
	if errors.Is(err, gitstate.ErrNotRepository) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	files := []string{}
	if len(dirty) == 0 {
		return files, nil
	}
	for rel := range planPaths(root, dirs, indexFile) {
		if _, ok := dirty[rel]; ok {
			files = append(files, rel)
		}
	}
	sort.Strings(files)
	return files, nil
}

// planPaths returns the repository-relative paths of the plan files and index
// files that exist under dirs.
func planPaths(root string, dirs []string, indexFile string) map[string]struct{} {
	set := map[string]struct{}{}
	scanned, _ := scan.ScanRepository(root, dirs)
	for _, f := range scanned {
		if _, ok := format.Parse(path.Base(f.RelPath), f.Content); ok {
			set[f.RelPath] = struct{}{}
		}
	}
	if indexFile == "" {
		return set
	}
	for _, dir := range dirs {
		if cleaned, err := scan.ValidateDirectory(dir); err == nil {
			set[path.Join(cleaned, indexFile)] = struct{}{}
		}
	}
	return set
}

// loadDirty reads the git state of every local repository once per pass. A
// repository that is not a git working tree gets no entry. One whose status
// cannot be read gets no entry either, is recorded in dirtyFailed, and is
// reported; neither stops the pass.
func (p *pass) loadDirty(ctx context.Context) {
	for _, repoID := range sortedKeys(p.roots) {
		dirty, err := gitstate.Dirty(ctx, p.roots[repoID], p.cfg.Directories)
		if errors.Is(err, gitstate.ErrNotRepository) {
			continue
		}
		if err != nil {
			p.svc.logger.Warn("plan file git state could not be read",
				zap.String("repository_id", repoID), zap.Error(err))
			p.dirtyFailed[repoID] = struct{}{}
			p.addError(repoID, "", ReasonGitStatusFailed)
			continue
		}
		p.dirty[repoID] = dirty
	}
}

// localRepositories returns the workspace's local repositories sorted by name.
func (s *Service) localRepositories(ctx context.Context, workspaceID string) ([]*taskmodels.Repository, error) {
	repos, err := s.tasks.ListRepositories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	local := make([]*taskmodels.Repository, 0, len(repos))
	for _, repo := range repos {
		if repo.SourceType == localSourceType && repo.LocalPath != "" {
			local = append(local, repo)
		}
	}
	sort.SliceStable(local, func(i, j int) bool {
		if local[i].Name != local[j].Name {
			return local[i].Name < local[j].Name
		}
		return local[i].ID < local[j].ID
	})
	return local, nil
}

// GitStatus lists, per local repository of the workspace, the plan files and
// plan indexes with uncommitted changes. Repositories that are not git working
// trees, or whose state cannot be read, are listed with no files.
func (s *Service) GitStatus(ctx context.Context, workspaceID string) ([]GitRepoStatus, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return nil, err
	}
	if s.tasks == nil {
		return nil, errSyncNotWired
	}
	unlock := s.LockWorkspace(workspaceID)
	defer unlock()
	dirs, indexFile, err := s.gitScope(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	repos, err := s.localRepositories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]GitRepoStatus, 0, len(repos))
	for _, repo := range repos {
		files, err := planDirtyFiles(ctx, repo.LocalPath, dirs, indexFile)
		if err != nil {
			s.logger.Warn("plan file git status failed",
				zap.String("repository_id", repo.ID), zap.Error(err))
			files = []string{}
		}
		result = append(result, GitRepoStatus{RepositoryID: repo.ID, RepositoryName: repo.Name, Files: files})
	}
	return result, nil
}

func gitFailure(code, message, output string) *GitError {
	return &GitError{Code: code, Message: message, Output: output}
}

// CommitPlans creates one commit on the repository's current branch that holds
// exactly the plan files and plan indexes with uncommitted changes. Other
// changes, staged ones included, stay out of the commit. It never pushes. A
// repository in the middle of a merge, rebase, cherry-pick, or revert is
// refused, and a failed commit leaves the working tree and index as they were.
func (s *Service) CommitPlans(ctx context.Context, workspaceID string, req CommitRequest) (CommitResult, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return CommitResult{}, err
	}
	if s.tasks == nil {
		return CommitResult{}, errSyncNotWired
	}
	unlock := s.LockWorkspace(workspaceID)
	defer unlock()
	result, err := s.commitLocked(ctx, workspaceID, req)
	var gitErr *GitError
	switch {
	case err == nil:
		incCommit(s.logger, commitCommitted)
		s.refreshAfterCommit(ctx, workspaceID)
	case errors.As(err, &gitErr) && gitErr.Code != CodeRepositoryNotFound:
		incCommit(s.logger, gitErr.Code)
	}
	return result, err
}

func (s *Service) commitLocked(ctx context.Context, workspaceID string, req CommitRequest) (CommitResult, error) {
	repos, err := s.localRepositories(ctx, workspaceID)
	if err != nil {
		return CommitResult{}, err
	}
	var root string
	for _, repo := range repos {
		if repo.ID == req.RepositoryID {
			root = repo.LocalPath
		}
	}
	if root == "" {
		return CommitResult{}, gitFailure(CodeRepositoryNotFound, "the repository is not a local repository of the workspace", "")
	}
	if gitstate.Busy(ctx, root) {
		return CommitResult{}, gitFailure(CodeRepositoryBusy,
			"the repository is in the middle of a merge, rebase, cherry-pick, or revert", "")
	}
	dirs, indexFile, err := s.gitScope(ctx, workspaceID)
	if err != nil {
		return CommitResult{}, err
	}
	files, err := planDirtyFiles(ctx, root, dirs, indexFile)
	if err != nil {
		return CommitResult{}, err
	}
	if len(files) == 0 {
		return CommitResult{}, gitFailure(CodeNothingToCommit, "no plan file has uncommitted changes", "")
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = DefaultCommitMessage
	}
	sha, output, err := gitstate.CommitPaths(ctx, root, files, message)
	if err != nil {
		if strings.TrimSpace(output) == "" {
			output = err.Error()
		}
		return CommitResult{}, gitFailure(CodeCommitFailed, "git could not create the commit", tailLines(output, maxCommitOutputLines))
	}
	return CommitResult{Commit: sha, Files: files}, nil
}

// refreshAfterCommit runs a pass so the uncommitted flags clear. The commit
// already exists, so a failing pass is only logged.
func (s *Service) refreshAfterCommit(ctx context.Context, workspaceID string) {
	if _, err := s.runPass(ctx, workspaceID); err != nil {
		s.logger.Warn("plan file sync after commit failed", zap.String("workspace_id", workspaceID), zap.Error(err))
	}
}

// tailLines returns the last n non-empty lines of text.
func tailLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
