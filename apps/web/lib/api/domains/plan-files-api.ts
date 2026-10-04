import { ApiError, fetchJson, type ApiRequestOptions } from "../client";

// Wire contract of apps/backend/internal/planfiles (handlers.go, models.go,
// sync_types.go). Status values are protocol tokens: they are sent verbatim and
// translated only when displayed.

/** Board statuses a step can be assigned to. `hidden` is never mapped. */
export const PLAN_FILE_STATUSES = [
  "queued",
  "in_progress",
  "waiting_owner",
  "waiting_external",
  "deferred",
  "done",
] as const;

export type PlanFileStatus = (typeof PLAN_FILE_STATUSES)[number];

export type PlanFileStatusSteps = Partial<Record<PlanFileStatus, string>>;

export type PlanFilePassCounts = {
  created: number;
  updated: number;
  moved: number;
  archived: number;
  unarchived: number;
  failed: number;
};

export type PlanFileErrorRow = {
  repository_id: string;
  repository_name: string;
  rel_path: string;
  reason: string;
};

export type PlanFilesConfig = {
  workspace_id: string;
  enabled: boolean;
  workflow_id: string;
  status_steps: PlanFileStatusSteps;
  directories: string[];
  /** Step ID to executor name. */
  executor_steps: Record<string, string>;
  notes_heading: string;
  wake_on_date: boolean;
  /** 0 turns the stale flag off. */
  stale_after_days: number;
  /** Empty means no index file. */
  index_file: string;
  last_pass_at?: string;
  last_pass_ok: boolean;
  last_counts: PlanFilePassCounts;
  last_file_errors: PlanFileErrorRow[];
  created_at: string;
  updated_at: string;
};

export type PutPlanFilesConfigRequest = {
  enabled: boolean;
  workflow_id: string;
  status_steps: PlanFileStatusSteps;
  directories: string[];
  /** Omitted fields keep the stored value. */
  executor_steps?: Record<string, string>;
  notes_heading?: string;
  wake_on_date?: boolean;
  stale_after_days?: number;
  index_file?: string;
};

export type CreatePlanBoardResult = {
  workflow_id: string;
  status_steps: PlanFileStatusSteps;
};

export type PlanFilePassSummary = {
  outcome: "ok" | "partial" | "failed" | "skipped";
  at: string;
  counts: PlanFilePassCounts;
  file_errors: PlanFileErrorRow[];
};

type PlanFilesApiOptions = ApiRequestOptions & { workspaceId: string };

function planFilesUrl(path: string, workspaceId: string): string {
  return `/api/v1/plan-files/${path}?workspace_id=${encodeURIComponent(workspaceId)}`;
}

function requestOptions(options: PlanFilesApiOptions): ApiRequestOptions {
  const { workspaceId: _workspaceId, ...rest } = options;
  return rest;
}

function withMethod(
  options: PlanFilesApiOptions,
  method: string,
  body?: unknown,
): ApiRequestOptions {
  const rest = requestOptions(options);
  return {
    ...rest,
    init: {
      ...(rest.init ?? {}),
      method,
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    },
  };
}

/** Resolves to null when the workspace has no plan-files config yet (404). */
export async function getPlanFilesConfig(
  options: PlanFilesApiOptions,
): Promise<PlanFilesConfig | null> {
  try {
    return await fetchJson<PlanFilesConfig>(
      planFilesUrl("config", options.workspaceId),
      requestOptions(options),
    );
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}

export function putPlanFilesConfig(
  payload: PutPlanFilesConfigRequest,
  options: PlanFilesApiOptions,
): Promise<PlanFilesConfig> {
  return fetchJson<PlanFilesConfig>(
    planFilesUrl("config", options.workspaceId),
    withMethod(options, "PUT", payload),
  );
}

/** Creates a workflow from the built-in Plans template; the config is not saved. */
export function createPlanFilesBoard(options: PlanFilesApiOptions): Promise<CreatePlanBoardResult> {
  return fetchJson<CreatePlanBoardResult>(
    planFilesUrl("board", options.workspaceId),
    withMethod(options, "POST"),
  );
}

/** Runs one pass now. Rejects with ApiError status 409 when a pass is already running. */
export function syncPlanFilesNow(options: PlanFilesApiOptions): Promise<PlanFilePassSummary> {
  return fetchJson<PlanFilePassSummary>(
    planFilesUrl("sync", options.workspaceId),
    withMethod(options, "POST"),
  );
}
