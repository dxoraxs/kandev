import { ApiError, fetchJson } from "../client";

/** Kinds the backend's repository maintenance launcher accepts. */
export type RepositoryMaintenanceKind = "plan_adaptation" | "repository_cleanup";

export type RepositoryMaintenanceStart = {
  task_id: string;
  /** Empty when the task exists but a previous launch did not create a session. */
  session_id: string;
  /** True when an active task of this kind already existed for the repository. */
  existing: boolean;
};

/** Closed set of reasons a start can be refused or fail; the UI maps each to copy. */
export type MaintenanceFailureReason =
  | "no_agent_profile"
  | "no_workflow"
  | "repository_not_local"
  | "kind_unavailable"
  | "repository_not_found"
  | "failed";

const REFUSAL_REASONS: readonly MaintenanceFailureReason[] = [
  "no_agent_profile",
  "no_workflow",
  "repository_not_local",
  "kind_unavailable",
];

export class MaintenanceTaskError extends Error {
  readonly reason: MaintenanceFailureReason;
  /** Set when the task was created before the failure (a launch failure). */
  readonly taskId?: string;

  constructor(reason: MaintenanceFailureReason, message: string, taskId?: string) {
    super(message);
    this.name = "MaintenanceTaskError";
    this.reason = reason;
    this.taskId = taskId;
  }
}

function bodyField(body: unknown, field: string): string | undefined {
  if (!body || typeof body !== "object") return undefined;
  const value = (body as Record<string, unknown>)[field];
  return typeof value === "string" && value !== "" ? value : undefined;
}

function toMaintenanceError(err: ApiError): MaintenanceTaskError {
  const reason = bodyField(err.body, "reason");
  const known = REFUSAL_REASONS.find((candidate) => candidate === reason);
  if (known) return new MaintenanceTaskError(known, err.message);
  if (err.status === 404) return new MaintenanceTaskError("repository_not_found", err.message);
  return new MaintenanceTaskError("failed", err.message, bodyField(err.body, "task_id"));
}

/**
 * Starts (or finds the active) maintenance task of a kind for a repository.
 * Rejects with a MaintenanceTaskError carrying a typed reason; backend text is
 * never meant for display.
 */
export async function startRepositoryMaintenanceTask(
  repositoryId: string,
  kind: RepositoryMaintenanceKind,
): Promise<RepositoryMaintenanceStart> {
  try {
    return await fetchJson<RepositoryMaintenanceStart>(
      `/api/v1/repositories/${encodeURIComponent(repositoryId)}/maintenance-tasks`,
      { init: { method: "POST", body: JSON.stringify({ kind }) } },
    );
  } catch (err) {
    if (err instanceof ApiError) throw toMaintenanceError(err);
    throw err;
  }
}
