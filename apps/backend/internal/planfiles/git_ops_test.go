package planfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gitStatusURL = "/api/v1/plan-files/git-status?workspace_id=" + testWorkspace
	commitURL    = "/api/v1/plan-files/commit?workspace_id=" + testWorkspace

	gitPlanQueued   = "docs/plans/queued.md"
	gitPlanNew      = "docs/plans/new plan.md"
	gitUnrelated    = "notes/unrelated.txt"
	gitStagedFile   = "notes/staged.txt"
	gitDefaultTitle = "docs(plans): update plan files"
)

func (h *syncHarness) git(args ...string) string {
	h.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = h.root
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	require.NoError(h.t, err, "git %s: %s", strings.Join(args, " "), out)
	return string(out)
}

// initGit turns the harness repository into a git repository with one commit
// holding a plan file, an unrelated file, and a file that is later staged.
func (h *syncHarness) initGit() {
	h.t.Helper()
	h.git("init", "-q", "-b", "main")
	h.git("config", "user.name", "Test")
	h.git("config", "user.email", "test@example.com")
	h.git("config", "commit.gpgsign", "false")
	h.writeFile(gitPlanQueued, planDoc("queued", "Queued"))
	h.writeFile(gitUnrelated, "u\n")
	h.writeFile(gitStagedFile, "s\n")
	h.git("add", "-A")
	h.git("commit", "-q", "-m", "initial")
}

// dirtyFourFiles leaves the scenario of the commit acceptance: a modified plan
// file, an untracked plan file, an unrelated modified file, and an unrelated
// staged file.
func (h *syncHarness) dirtyFourFiles() {
	h.t.Helper()
	h.writeFile(gitPlanQueued, planDoc("queued", "Queued", "owner: me"))
	h.writeFile(gitPlanNew, planDoc("queued", "New"))
	h.writeFile(gitUnrelated, "u2\n")
	h.writeFile(gitStagedFile, "s2\n")
	h.git("add", gitStagedFile)
}

type gitRepoBody struct {
	RepositoryID   string   `json:"repository_id"`
	RepositoryName string   `json:"repository_name"`
	Files          []string `json:"files"`
}

type gitStatusBody struct {
	Repositories []gitRepoBody `json:"repositories"`
}

type commitBody struct {
	Commit string   `json:"commit"`
	Files  []string `json:"files"`
	Error  string   `json:"error"`
	Code   string   `json:"code"`
	Output string   `json:"output"`
}

func getGitStatus(t *testing.T, h *syncHarness) gitStatusBody {
	t.Helper()
	rec := doRequest(t, newTestRouter(t, h.svc), http.MethodGet, gitStatusURL, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out gitStatusBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func postCommit(t *testing.T, h *syncHarness, body any) (int, commitBody) {
	t.Helper()
	rec := doRequest(t, newTestRouter(t, h.svc), http.MethodPost, commitURL, body)
	var out commitBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return rec.Code, out
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.2
func TestHandlers_GitStatusListsOnlyPlanFilesAndTheIndex(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	idx := "INDEX.md"
	req := validRequest()
	req.IndexFile = &idx
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(t, err)
	h.dirtyFourFiles()
	h.writeFile("docs/plans/INDEX.md", "# generated\n")
	h.writeFile("docs/plans/loose.md", "# not a plan file\n")

	body := getGitStatus(t, h)

	require.Len(t, body.Repositories, 1)
	assert.Equal(t, testRepoID, body.Repositories[0].RepositoryID)
	assert.Equal(t, testRepoName, body.Repositories[0].RepositoryName)
	assert.Equal(t, []string{"docs/plans/INDEX.md", gitPlanNew, gitPlanQueued}, body.Repositories[0].Files)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.2
func TestHandlers_GitStatusIsEmptyForARepositoryThatIsNotAGitWorkingTree(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(gitPlanQueued, planDoc("queued", "Queued"))

	rec := doRequest(t, newTestRouter(t, h.svc), http.MethodGet, gitStatusURL, nil)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"files":[]`)
	var body gitStatusBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Repositories, 1)
	assert.Empty(t, body.Repositories[0].Files)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.3
func TestHandlers_CommitCommitsExactlyThePlanFiles(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.dirtyFourFiles()

	status, body := postCommit(t, h, map[string]string{"repository_id": testRepoID})

	require.Equal(t, http.StatusOK, status, body.Error)
	assert.Equal(t, []string{gitPlanNew, gitPlanQueued}, body.Files)
	assert.Equal(t, strings.TrimSpace(h.git("rev-parse", "HEAD")), body.Commit)
	assert.Equal(t, gitDefaultTitle, strings.TrimSpace(h.git("log", "-1", "--format=%s")))
	committed := strings.Split(strings.Trim(h.git("show", "--name-only", "--format=", "-z", "HEAD"), "\x00\n"), "\x00")
	assert.ElementsMatch(t, []string{gitPlanNew, gitPlanQueued}, committed)
	assert.Equal(t, "M  notes/staged.txt\n M notes/unrelated.txt\n", h.git("status", "--porcelain"))
	assert.Empty(t, getGitStatus(t, h).Repositories[0].Files)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.3
func TestHandlers_CommitUsesTheGivenMessage(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.dirtyFourFiles()

	status, _ := postCommit(t, h, map[string]string{"repository_id": testRepoID, "message": "  plans: tidy up  "})

	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "plans: tidy up", strings.TrimSpace(h.git("log", "-1", "--format=%s")))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.4
func TestHandlers_CommitRefusesARepositoryInTheMiddleOfAMerge(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.dirtyFourFiles()
	require.NoError(t, os.WriteFile(filepath.Join(h.root, ".git", "MERGE_HEAD"),
		[]byte(strings.TrimSpace(h.git("rev-parse", "HEAD"))+"\n"), 0o644))
	before := h.git("status", "--porcelain")

	status, body := postCommit(t, h, map[string]string{"repository_id": testRepoID})

	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, CodeRepositoryBusy, body.Code)
	assert.Equal(t, before, h.git("status", "--porcelain"))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.4
func TestHandlers_CommitRejectedByAHookReportsTheOutputAndChangesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.dirtyFourFiles()
	hook := "#!/bin/sh\necho 'plan lint failed' >&2\nexit 1\n"
	require.NoError(t, os.MkdirAll(filepath.Join(h.root, ".git", "hooks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(h.root, ".git", "hooks", "pre-commit"), []byte(hook), 0o755))
	before := h.git("status", "--porcelain")
	headBefore := h.git("rev-parse", "HEAD")

	status, body := postCommit(t, h, map[string]string{"repository_id": testRepoID})

	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Equal(t, CodeCommitFailed, body.Code)
	assert.Contains(t, body.Output, "plan lint failed")
	assert.Equal(t, before, h.git("status", "--porcelain"))
	assert.Equal(t, headBefore, h.git("rev-parse", "HEAD"))
}

func TestHandlers_CommitOutputKeepsOnlyTheLastLines(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.dirtyFourFiles()
	hook := "#!/bin/sh\nfor i in $(seq 1 60); do echo \"line $i\" >&2; done\nexit 1\n"
	require.NoError(t, os.MkdirAll(filepath.Join(h.root, ".git", "hooks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(h.root, ".git", "hooks", "pre-commit"), []byte(hook), 0o755))

	_, body := postCommit(t, h, map[string]string{"repository_id": testRepoID})

	assert.Contains(t, body.Output, "line 60")
	assert.NotContains(t, body.Output, "line 1\n")
	assert.LessOrEqual(t, strings.Count(body.Output, "\n"), maxCommitOutputLines)
}

func TestHandlers_CommitWithNothingToCommit(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.writeFile(gitUnrelated, "u2\n")

	status, body := postCommit(t, h, map[string]string{"repository_id": testRepoID})

	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, CodeNothingToCommit, body.Code)
}

func TestHandlers_CommitOnARepositoryThatIsNotAGitWorkingTreeHasNothingToCommit(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(gitPlanQueued, planDoc("queued", "Queued"))

	status, body := postCommit(t, h, map[string]string{"repository_id": testRepoID})

	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, CodeNothingToCommit, body.Code)
}

func TestHandlers_CommitErrors(t *testing.T) {
	t.Run("unknown repository", func(t *testing.T) {
		h := newSyncHarness(t)
		status, body := postCommit(t, h, map[string]string{"repository_id": "nope"})
		assert.Equal(t, http.StatusNotFound, status)
		assert.Equal(t, CodeRepositoryNotFound, body.Code)
	})
	t.Run("missing repository id", func(t *testing.T) {
		h := newSyncHarness(t)
		status, _ := postCommit(t, h, map[string]string{})
		assert.Equal(t, http.StatusBadRequest, status)
	})
	t.Run("malformed body", func(t *testing.T) {
		h := newSyncHarness(t)
		rec := doRequest(t, newTestRouter(t, h.svc), http.MethodPost, commitURL, "{")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing workspace", func(t *testing.T) {
		h := newSyncHarness(t)
		rec := doRequest(t, newTestRouter(t, h.svc), http.MethodPost, "/api/v1/plan-files/commit", nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		rec = doRequest(t, newTestRouter(t, h.svc), http.MethodGet, "/api/v1/plan-files/git-status", nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.1
func TestPass_KeepsTheDirtySetPerRepositoryAndIgnoresNonRepositories(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.dirtyFourFiles()
	cfg, err := h.svc.store.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)

	p := newPass(h.svc, cfg, time.Now().UTC())
	require.NoError(t, p.run(context.Background()))

	assert.Contains(t, p.dirty[testRepoID], gitPlanQueued)
	assert.Contains(t, p.dirty[testRepoID], gitPlanNew)

	plain := newSyncHarness(t)
	plain.writeFile(gitPlanQueued, planDoc("queued", "Queued"))
	cfg, err = plain.svc.store.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	p = newPass(plain.svc, cfg, time.Now().UTC())
	require.NoError(t, p.run(context.Background()))
	assert.Empty(t, p.dirty[testRepoID])
	assert.Equal(t, OutcomeOK, p.summary(nil).Outcome)
}
