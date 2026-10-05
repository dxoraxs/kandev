package planfiles

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const localSourceType = "local"

// errIdentityConflict reports that two plan files resolved to one plan task.
var errIdentityConflict = errors.New("another plan file already owns this task")

// passFailure marks an error that ends the whole pass, with the reason shown
// in the settings section.
type passFailure struct {
	reason string
	err    error
}

func (f *passFailure) Error() string { return f.reason + ": " + f.err.Error() }
func (f *passFailure) Unwrap() error { return f.err }

// planEntry is one parsed plan file of a local repository.
type planEntry struct {
	repo    *taskmodels.Repository
	relPath string
	file    format.PlanFile
	extID   string
}

func (e planEntry) key() string { return pathKey(e.repo.ID, e.relPath) }

func (e planEntry) orderKey() orderKey { return newOrderKey(e.file.Order, e.relPath, e.repo.ID) }

func pathKey(repositoryID, relPath string) string { return repositoryID + "\x00" + relPath }

// pass is the state of one sync pass over a workspace.
type pass struct {
	svc *Service
	cfg *Config
	now time.Time

	counts PassCounts
	// unadapted tallies files without a board key per repository ID.
	unadapted map[string]int
	errs      []FileErrorRow
	degraded  bool

	rows       []*TaskRow
	rowsByPath map[string]*TaskRow
	rowsByExt  map[string]*TaskRow
	repoNames  map[string]string

	// seen holds every plan file path found this pass, hidden or failed ones
	// included: a task is archived only when its file is truly gone.
	seen map[string]struct{}
	// protectedRepos and protectedPaths cover files that could not be read.
	// Their tasks are left untouched rather than archived.
	protectedRepos map[string]struct{}
	protectedPaths map[string]struct{}
	// claimed holds the tasks a plan file already resolved to this pass.
	claimed map[string]struct{}
	// pathTask maps a plan file path to its task for dependency links.
	pathTask map[string]string
	// keys holds the desired order key of every task synced this pass.
	keys map[string]orderKey
	// roots maps a local repository ID to its path.
	roots map[string]string
	// volatile holds the tasks whose position on the board this pass cannot
	// read as a person's reorder: new, changed, moved, or moved by a person.
	volatile map[string]struct{}
	// dirty holds, per local repository ID, the paths under the scanned
	// directories that are modified, staged, or untracked. A repository that
	// is not a git working tree has no entry.
	dirty map[string]map[string]struct{}
	// finalStep maps a task to the board step it holds after this pass applied
	// its file, so the index groups plans by where they are on the board.
	finalStep map[string]string
}

func newPass(svc *Service, cfg *Config, now time.Time) *pass {
	return &pass{
		svc: svc, cfg: cfg, now: now,
		unadapted:  map[string]int{},
		rowsByPath: map[string]*TaskRow{}, rowsByExt: map[string]*TaskRow{}, repoNames: map[string]string{},
		seen: map[string]struct{}{}, protectedRepos: map[string]struct{}{}, protectedPaths: map[string]struct{}{},
		claimed: map[string]struct{}{}, pathTask: map[string]string{}, keys: map[string]orderKey{},
		roots: map[string]string{}, volatile: map[string]struct{}{}, dirty: map[string]map[string]struct{}{},
		finalStep: map[string]string{},
	}
}

func (p *pass) summary(runErr error) PassSummary {
	var failure *passFailure
	outcome := OutcomeOK
	switch {
	case errors.As(runErr, &failure):
		outcome = OutcomeFailed
		p.errs = append(p.errs, FileErrorRow{Reason: failure.reason})
		p.svc.logger.Warn("plan file pass failed", zap.String("reason", failure.reason), zap.Error(failure.err))
	case runErr != nil:
		outcome = OutcomeFailed
		p.errs = append(p.errs, FileErrorRow{Reason: ReasonTaskService})
		p.svc.logger.Warn("plan file pass failed", zap.Error(runErr))
	case len(p.errs) > 0 || p.degraded:
		outcome = OutcomePartial
	}
	errs := p.errs
	if errs == nil {
		errs = []FileErrorRow{}
	}
	return PassSummary{Outcome: outcome, At: p.now, Counts: p.counts, FileErrors: errs}
}

func (p *pass) run(ctx context.Context) error {
	if _, err := p.svc.workflows.GetWorkflow(ctx, p.cfg.WorkflowID); err != nil {
		return &passFailure{reason: ReasonWorkflowMissing, err: err}
	}
	repos, err := p.svc.tasks.ListRepositories(ctx, p.cfg.WorkspaceID)
	if err != nil {
		return &passFailure{reason: ReasonRepositoryList, err: err}
	}
	if err := p.loadRows(ctx); err != nil {
		return &passFailure{reason: ReasonTaskService, err: err}
	}
	entries := p.collect(repos)
	p.loadDirty(ctx)
	entries = p.rejectDuplicates(entries)
	tracked := p.resolveAll(ctx, entries)
	var deps []depWork
	for _, tr := range tracked {
		if row := p.applyTracked(ctx, tr); row != nil {
			deps = append(deps, depWork{entry: tr.entry, row: row})
		}
	}
	p.syncDependencies(ctx, deps)
	p.archiveMissing(ctx)
	p.reorder(ctx)
	p.writeIndexes(ctx, tracked)
	return nil
}

func (p *pass) loadRows(ctx context.Context) error {
	rows, err := p.svc.store.ListTaskRows(ctx, p.cfg.WorkspaceID)
	if err != nil {
		return err
	}
	p.rows = rows
	for _, row := range rows {
		p.rowsByPath[pathKey(row.RepositoryID, row.RelPath)] = row
		p.rowsByExt[row.ExternalID] = row
	}
	return nil
}

// collect scans every local repository and parses its plan files in a stable
// order. Files that are not plan files are counted as unadapted and otherwise
// dropped; scan problems and parse errors become file errors without stopping
// the pass.
func (p *pass) collect(repos []*taskmodels.Repository) []planEntry {
	var entries []planEntry
	for _, repo := range repos {
		p.repoNames[repo.ID] = repo.Name
		if repo.SourceType != localSourceType || repo.LocalPath == "" {
			p.protectedRepos[repo.ID] = struct{}{}
			continue
		}
		p.roots[repo.ID] = repo.LocalPath
		files, scanErrs := scan.ScanRepository(repo.LocalPath, p.cfg.Directories)
		p.recordScanErrors(repo, scanErrs)
		for _, f := range files {
			if isIndexFile(p.cfg.IndexFile, f.RelPath) {
				continue
			}
			pf, ok := format.Parse(path.Base(f.RelPath), f.Content)
			if !ok {
				p.counts.Unadapted++
				p.unadapted[repo.ID]++
				continue
			}
			entry := planEntry{repo: repo, relPath: f.RelPath, file: pf, extID: pf.ExternalID}
			if entry.extID == "" {
				entry.extID = defaultExternalID(repo.ID, f.RelPath)
			}
			if len(pf.ParseErrors) > 0 {
				p.addError(entry.repo.ID, entry.relPath, ReasonParse)
			}
			entries = append(entries, entry)
		}
	}
	return entries
}

func (p *pass) recordScanErrors(repo *taskmodels.Repository, scanErrs []scan.FileError) {
	for _, fe := range scanErrs {
		p.addError(repo.ID, fe.RelPath, string(fe.Reason))
		if fe.RelPath == "" {
			p.protectedRepos[repo.ID] = struct{}{}
			continue
		}
		p.protectedPaths[pathKey(repo.ID, fe.RelPath)] = struct{}{}
	}
}

func (p *pass) addError(repositoryID, relPath, reason string) {
	p.errs = append(p.errs, FileErrorRow{
		RepositoryID: repositoryID, RepositoryName: p.repoNames[repositoryID], RelPath: relPath, Reason: reason,
	})
}

// isProtected reports that the file behind a row could not be read this pass.
func (p *pass) isProtected(row *TaskRow) bool {
	return p.pathUnreadable(row.RepositoryID, row.RelPath)
}

// pathUnreadable reports that the file, or its directory or repository, could
// not be read this pass.
func (p *pass) pathUnreadable(repositoryID, relPath string) bool {
	if _, ok := p.protectedRepos[repositoryID]; ok {
		return true
	}
	if _, ok := p.protectedPaths[pathKey(repositoryID, relPath)]; ok {
		return true
	}
	_, ok := p.protectedPaths[pathKey(repositoryID, path.Dir(relPath))]
	return ok
}

// rejectDuplicates drops every plan file whose external identifier another
// plan file of the workspace also resolves to, and reports each of them. Their
// tasks, if any, are neither updated nor archived.
func (p *pass) rejectDuplicates(entries []planEntry) []planEntry {
	count := make(map[string]int, len(entries))
	for _, e := range entries {
		count[e.extID]++
	}
	unique := make([]planEntry, 0, len(entries))
	for _, e := range entries {
		if count[e.extID] == 1 {
			unique = append(unique, e)
			continue
		}
		p.seen[e.key()] = struct{}{}
		p.addError(e.repo.ID, e.relPath, ReasonDuplicateExternal)
	}
	return unique
}

// fail records a per-file failure of the task system.
func (p *pass) fail(repositoryID, relPath string, err error) {
	reason := ReasonTaskService
	if errors.Is(err, errIdentityConflict) {
		reason = ReasonDuplicateExternal
	}
	p.counts.Failed++
	p.addError(repositoryID, relPath, reason)
	p.svc.logger.Warn("plan file task sync failed",
		zap.String("repository_id", repositoryID), zap.String("rel_path", relPath), zap.Error(err))
}

func (p *pass) stepFor(status format.BoardStatus) (string, error) {
	stepID := p.cfg.StatusSteps[status]
	if stepID == "" {
		return "", fmt.Errorf("no step is mapped to status %q", status)
	}
	return stepID, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
