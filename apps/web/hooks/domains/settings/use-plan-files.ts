import { useCallback, useEffect, useMemo, useState } from "react";
import { t } from "@/lib/i18n";
import { ApiError } from "@/lib/api/client";
import { listWorkflows } from "@/lib/api/domains/kanban-api";
import { listWorkflowSteps } from "@/lib/api/domains/workflow-api";
import {
  PLAN_FILE_STATUSES,
  createPlanFilesBoard,
  getPlanFilesConfig,
  putPlanFilesConfig,
  syncPlanFilesNow,
  type PlanFileStatusSteps,
  type PlanFilesConfig,
} from "@/lib/api/domains/plan-files-api";

// Directories offered before a config exists. The backend returns the stored
// (initially these) directories as soon as a config is saved.
export const DEFAULT_PLAN_DIRECTORIES = ["docs/plans", "docs/superpowers/plans"];

export type ExecutorRow = { stepId: string; name: string };

export type PlanFilesDraft = {
  enabled: boolean;
  workflowId: string;
  statusSteps: PlanFileStatusSteps;
  directories: string[];
  executorRows: ExecutorRow[];
  notesHeading: string;
  wakeOnDate: boolean;
  /** Kept as typed text so the field can be empty while editing. */
  staleAfterDays: string;
  indexFile: string;
};

const EXECUTOR_NAME_MAX_BYTES = 40;
const STALE_AFTER_DAYS_MAX = 365;

export type PlanFilesStep = { id: string; name: string };
export type PlanFilesWorkflow = { id: string; name: string };

/** One of the two non-error outcomes a Sync now can leave inline. */
export type SyncNotice = "running" | null;

const EMPTY_DRAFT: PlanFilesDraft = {
  enabled: false,
  workflowId: "",
  statusSteps: {},
  directories: DEFAULT_PLAN_DIRECTORIES,
  executorRows: [],
  notesHeading: "",
  wakeOnDate: true,
  staleAfterDays: "7",
  indexFile: "",
};

function draftFromConfig(config: PlanFilesConfig | null): PlanFilesDraft {
  if (!config) return EMPTY_DRAFT;
  return {
    enabled: config.enabled,
    workflowId: config.workflow_id,
    statusSteps: { ...config.status_steps },
    directories: [...config.directories],
    executorRows: Object.entries(config.executor_steps ?? {}).map(([stepId, name]) => ({
      stepId,
      name,
    })),
    notesHeading: config.notes_heading ?? "",
    wakeOnDate: config.wake_on_date ?? true,
    staleAfterDays: String(config.stale_after_days ?? 7),
    indexFile: config.index_file ?? "",
  };
}

function parseStaleAfterDays(value: string): number | null {
  if (!/^\d+$/.test(value.trim())) return null;
  const days = Number(value);
  return days <= STALE_AFTER_DAYS_MAX ? days : null;
}

function executorRowComplete(row: ExecutorRow): boolean {
  const bytes = new TextEncoder().encode(row.name.trim()).length;
  return row.stepId !== "" && bytes > 0 && bytes <= EXECUTOR_NAME_MAX_BYTES;
}

function executorStepsPayload(rows: ExecutorRow[]): Record<string, string> {
  return Object.fromEntries(rows.map((row) => [row.stepId, row.name.trim()]));
}

function sameRows(a: ExecutorRow[], b: ExecutorRow[]): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/**
 * Save needs a board, a step for every status except hidden, a directory,
 * complete executor rows, and a valid stale threshold.
 */
export function isDraftComplete(draft: PlanFilesDraft): boolean {
  return (
    draft.workflowId !== "" &&
    draft.directories.length > 0 &&
    draft.executorRows.every(executorRowComplete) &&
    parseStaleAfterDays(draft.staleAfterDays) !== null &&
    PLAN_FILE_STATUSES.every((status) => Boolean(draft.statusSteps[status]))
  );
}

function sameDraft(a: PlanFilesDraft, b: PlanFilesDraft): boolean {
  return (
    a.enabled === b.enabled &&
    a.workflowId === b.workflowId &&
    a.directories.join("\0") === b.directories.join("\0") &&
    sameRows(a.executorRows, b.executorRows) &&
    a.notesHeading === b.notesHeading &&
    a.wakeOnDate === b.wakeOnDate &&
    a.staleAfterDays === b.staleAfterDays &&
    a.indexFile === b.indexFile &&
    PLAN_FILE_STATUSES.every(
      (status) => (a.statusSteps[status] ?? "") === (b.statusSteps[status] ?? ""),
    )
  );
}

function errorMessage(err: unknown): string {
  return err instanceof Error && err.message ? err.message : t("common:requestFailed");
}

async function fetchBoards(workspaceId: string): Promise<PlanFilesWorkflow[]> {
  const res = await listWorkflows(workspaceId);
  return res.workflows
    .filter((wf) => wf.style !== "office")
    .map((wf) => ({ id: wf.id, name: wf.name }));
}

function useBoardSteps(workflowId: string) {
  const [steps, setSteps] = useState<PlanFilesStep[]>([]);
  useEffect(() => {
    let cancelled = false;
    if (!workflowId) {
      setSteps([]);
      return;
    }
    listWorkflowSteps(workflowId)
      .then((res) => {
        if (cancelled) return;
        const ordered = [...res.steps].sort((a, b) => a.position - b.position);
        setSteps(ordered.map((step) => ({ id: step.id, name: step.name })));
      })
      .catch(() => {
        if (!cancelled) setSteps([]);
      });
    return () => {
      cancelled = true;
    };
  }, [workflowId]);
  return steps;
}

type ActionDeps = {
  workspaceId: string;
  setConfig: (config: PlanFilesConfig | null) => void;
  setError: (message: string | null) => void;
};

function useSaveConfig({
  draft,
  workspaceId,
  setConfig,
  setDraft,
  setError,
}: ActionDeps & { draft: PlanFilesDraft; setDraft: (draft: PlanFilesDraft) => void }) {
  const [saving, setSaving] = useState(false);
  const run = useCallback(async () => {
    setSaving(true);
    try {
      const saved = await putPlanFilesConfig(
        {
          enabled: draft.enabled,
          workflow_id: draft.workflowId,
          status_steps: draft.statusSteps,
          directories: draft.directories,
          executor_steps: executorStepsPayload(draft.executorRows),
          notes_heading: draft.notesHeading.trim(),
          wake_on_date: draft.wakeOnDate,
          stale_after_days: parseStaleAfterDays(draft.staleAfterDays) ?? 0,
          index_file: draft.indexFile.trim(),
        },
        { workspaceId },
      );
      setConfig(saved);
      setDraft(draftFromConfig(saved));
      setError(null);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  }, [draft, workspaceId, setConfig, setDraft, setError]);
  return { saving, run };
}

/** A 409 means a pass already holds the workspace lock: a notice, not an error. */
function useSyncNow({ workspaceId, setConfig, setError }: ActionDeps) {
  const [syncing, setSyncing] = useState(false);
  const [syncNotice, setSyncNotice] = useState<SyncNotice>(null);
  const run = useCallback(async () => {
    setSyncing(true);
    setSyncNotice(null);
    try {
      await syncPlanFilesNow({ workspaceId });
      setConfig(await getPlanFilesConfig({ workspaceId }));
      setError(null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) setSyncNotice("running");
      else setError(errorMessage(err));
    } finally {
      setSyncing(false);
    }
  }, [workspaceId, setConfig, setError]);
  return { syncing, syncNotice, run };
}

export function usePlanFiles(workspaceId: string) {
  const [config, setConfig] = useState<PlanFilesConfig | null>(null);
  const [draft, setDraft] = useState<PlanFilesDraft>(EMPTY_DRAFT);
  const [workflows, setWorkflows] = useState<PlanFilesWorkflow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creatingBoard, setCreatingBoard] = useState(false);
  const steps = useBoardSteps(draft.workflowId);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    Promise.all([getPlanFilesConfig({ workspaceId }), fetchBoards(workspaceId)])
      .then(([loaded, boards]) => {
        if (cancelled) return;
        setConfig(loaded);
        setDraft(draftFromConfig(loaded));
        setWorkflows(boards);
        setError(null);
      })
      .catch((err) => {
        if (!cancelled) setError(errorMessage(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId]);

  const patch = useCallback((changes: Partial<PlanFilesDraft>) => {
    setDraft((prev) => ({ ...prev, ...changes }));
    setError(null);
  }, []);

  const selectWorkflow = useCallback(
    (workflowId: string) => patch({ workflowId, statusSteps: {}, executorRows: [] }),
    [patch],
  );

  const setStatusStep = useCallback((status: keyof PlanFileStatusSteps, stepId: string) => {
    setDraft((prev) => ({
      ...prev,
      statusSteps: { ...prev.statusSteps, [status]: stepId },
      executorRows: prev.executorRows.filter((row) => row.stepId !== stepId),
    }));
    setError(null);
  }, []);

  const createBoard = useCallback(async () => {
    setCreatingBoard(true);
    try {
      const board = await createPlanFilesBoard({ workspaceId });
      setWorkflows(await fetchBoards(workspaceId));
      patch({
        workflowId: board.workflow_id,
        statusSteps: { ...board.status_steps },
        executorRows: [],
      });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setCreatingBoard(false);
    }
  }, [workspaceId, patch]);

  const save = useSaveConfig({ draft, workspaceId, setConfig, setDraft, setError });
  const sync = useSyncNow({ workspaceId, setConfig, setError });

  const dirty = useMemo(() => !sameDraft(draft, draftFromConfig(config)), [draft, config]);
  const complete = isDraftComplete(draft);

  return {
    config,
    draft,
    workflows,
    steps,
    loading,
    error,
    saving: save.saving,
    syncing: sync.syncing,
    creatingBoard,
    syncNotice: sync.syncNotice,
    complete,
    canSave: dirty && complete && !save.saving,
    canSync: Boolean(config?.enabled) && !sync.syncing,
    patch,
    selectWorkflow,
    setStatusStep,
    createBoard,
    save: save.run,
    syncNow: sync.run,
  };
}
