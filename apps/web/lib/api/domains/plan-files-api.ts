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
  /** Plan files whose header is not in the board format, counted by the last pass. */
  unadapted: number;
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

export type UnadaptedPlanFilesRepository = {
  repository_id: string;
  repository_name: string;
  count: number;
  directories: string[];
};

type PlanFilesApiOptions = ApiRequestOptions & { workspaceId: string };

function planFilesUrl(path: string, workspaceId: string): string {
  return `/api/v1/plan-files/${path}?workspace_id=${encodeURIComponent(workspaceId)}`;
}

function requestOptions(options: ApiRequestOptions & { workspaceId?: string }): ApiRequestOptions {
  const { workspaceId: _workspaceId, ...rest } = options;
  return rest;
}

function withMethod(
  options: ApiRequestOptions & { workspaceId?: string },
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

/**
 * Local repositories of the workspace that hold plan files without board
 * status, read live and independent of the sync config.
 */
export async function getUnadaptedPlanFiles(
  options: PlanFilesApiOptions & { repositoryId?: string },
): Promise<UnadaptedPlanFilesRepository[]> {
  const { repositoryId, ...rest } = options;
  const scope = repositoryId ? `&repository_id=${encodeURIComponent(repositoryId)}` : "";
  const res = await fetchJson<{ repositories: UnadaptedPlanFilesRepository[] | null }>(
    `${planFilesUrl("unadapted", options.workspaceId)}${scope}`,
    requestOptions(rest),
  );
  return res.repositories ?? [];
}

export type PlanDecisionBody = {
  action: "accept" | "return";
  /** Accept only: the status the plan moves to. The server defaults to done. */
  result?: "done" | "queued";
  comment?: string;
};

export type PlanDecisionErrorCode =
  | "not_plan_task"
  | "not_waiting_owner"
  | "file_changed"
  | "invalid_decision";

const PLAN_DECISION_ERROR_CODES: readonly string[] = [
  "not_plan_task",
  "not_waiting_owner",
  "file_changed",
  "invalid_decision",
];

/** The decision error code a rejected `decidePlan` carries, or null for any other failure. */
export function planDecisionErrorCode(err: unknown): PlanDecisionErrorCode | null {
  if (!(err instanceof ApiError)) return null;
  const code = (err.body as { code?: unknown } | null)?.code;
  return typeof code === "string" && PLAN_DECISION_ERROR_CODES.includes(code)
    ? (code as PlanDecisionErrorCode)
    : null;
}

/**
 * Records the owner's decision on a plan task waiting for them. The workspace
 * comes from the task, so the route takes no workspace parameter.
 */
export function decidePlan(
  taskId: string,
  body: PlanDecisionBody,
  options: ApiRequestOptions = {},
): Promise<{ board: PlanFileStatus }> {
  return fetchJson<{ board: PlanFileStatus }>(
    `/api/v1/plan-files/tasks/${encodeURIComponent(taskId)}/decision`,
    withMethod(options, "POST", body),
  );
}

export type PlanGitRepository = {
  repository_id: string;
  repository_name: string;
  /** Plan files and plan indexes with uncommitted changes. */
  files: string[];
};

export type PlanCommitBody = {
  repository_id: string;
  /** The server defaults to its plan-files commit message when empty. */
  message?: string;
};

export type PlanCommitResult = { commit: string; files: string[] };

export type PlanCommitErrorCode =
  | "repository_busy"
  | "nothing_to_commit"
  | "commit_failed"
  | "repository_not_found"
  | "invalid_commit";

const PLAN_COMMIT_ERROR_CODES: readonly string[] = [
  "repository_busy",
  "nothing_to_commit",
  "commit_failed",
  "repository_not_found",
  "invalid_commit",
];

// i18n-exempt: commit message data sent to git, not user-facing copy.
export const DEFAULT_PLAN_COMMIT_MESSAGE = "docs(plans): update plan files";

/** Local repositories of the workspace with the plan files that have uncommitted changes. */
export async function getPlanGitStatus(options: PlanFilesApiOptions): Promise<PlanGitRepository[]> {
  const res = await fetchJson<{ repositories: PlanGitRepository[] | null }>(
    planFilesUrl("git-status", options.workspaceId),
    requestOptions(options),
  );
  return res.repositories ?? [];
}

/** Commits the repository's uncommitted plan files; the server never pushes. */
export function commitPlanFiles(
  body: PlanCommitBody,
  options: PlanFilesApiOptions,
): Promise<PlanCommitResult> {
  return fetchJson<PlanCommitResult>(
    planFilesUrl("commit", options.workspaceId),
    withMethod(options, "POST", body),
  );
}

/**
 * The typed code and the last lines of git output a rejected `commitPlanFiles`
 * carries, or null for any other failure.
 */
export function planCommitError(
  err: unknown,
): { code: PlanCommitErrorCode; output: string } | null {
  if (!(err instanceof ApiError)) return null;
  const body = err.body as { code?: unknown; output?: unknown } | null;
  const code = body?.code;
  if (typeof code !== "string" || !PLAN_COMMIT_ERROR_CODES.includes(code)) return null;
  return {
    code: code as PlanCommitErrorCode,
    output: typeof body?.output === "string" ? body.output : "",
  };
}

export type CreatePlanBody = {
  repository_id: string;
  /** One of the configured plan directories. */
  directory: string;
  title: string;
  /** The server derives it from the title when empty. */
  file_name?: string;
  priority?: "critical" | "high" | "medium" | "low";
  executor?: string;
  body?: string;
};

export type CreatePlanResult = {
  task_id: string;
  repository_id: string;
  rel_path: string;
};

export type PlanCreateErrorCode = "file_exists" | "invalid_plan" | "repository_not_found";

const PLAN_CREATE_ERROR_CODES: readonly string[] = [
  "file_exists",
  "invalid_plan",
  "repository_not_found",
];

/**
 * Writes a new plan file and resolves once its task is on the board. Rejects
 * with an ApiError carrying a `planCreateErrorCode` when the file is refused.
 */
export function createPlan(
  body: CreatePlanBody,
  options: PlanFilesApiOptions,
): Promise<CreatePlanResult> {
  return fetchJson<CreatePlanResult>(
    planFilesUrl("plans", options.workspaceId),
    withMethod(options, "POST", body),
  );
}

/** The create error code a rejected `createPlan` carries, or null for any other failure. */
export function planCreateErrorCode(err: unknown): PlanCreateErrorCode | null {
  if (!(err instanceof ApiError)) return null;
  const code = (err.body as { code?: unknown } | null)?.code;
  return typeof code === "string" && PLAN_CREATE_ERROR_CODES.includes(code)
    ? (code as PlanCreateErrorCode)
    : null;
}
