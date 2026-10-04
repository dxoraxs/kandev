import type { EnsureTaskSessionStatus } from "@/hooks/domains/session/use-ensure-task-session";

export type SessionlessStatusLine = "none" | "preparing" | "unassigned";

export type SessionlessTaskViewModel = {
  /** Trimmed-non-empty description, or null to render the placeholder. */
  description: string | null;
  statusLine: SessionlessStatusLine;
  /** Workspace whose settings the unassigned notice links to, when known. */
  settingsWorkspaceId: string | null;
};

/**
 * Decides what a task with no session shows. Ensure errors render nothing here:
 * the page-level error banner owns them.
 */
export function resolveSessionlessTaskView(input: {
  description: string | null | undefined;
  status: EnsureTaskSessionStatus;
  workspaceId: string | null | undefined;
}): SessionlessTaskViewModel {
  const description = input.description?.trim() ? input.description : null;
  const statusLine = toStatusLine(input.status);
  const settingsWorkspaceId =
    statusLine === "unassigned" && input.workspaceId ? input.workspaceId : null;
  return { description, statusLine, settingsWorkspaceId };
}

function toStatusLine(status: EnsureTaskSessionStatus): SessionlessStatusLine {
  if (status === "preparing" || status === "unassigned") return status;
  return "none";
}
