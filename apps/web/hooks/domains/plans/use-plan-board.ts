import { useCallback, useEffect, useState } from "react";
import { useFeature } from "@/hooks/domains/features/use-feature";
import {
  getPlanFilesConfig,
  type PlanFileStatus,
  type PlanFilesConfig,
} from "@/lib/api/domains/plan-files-api";

// One read per workspace for the whole app: every task page asks the same
// question. A failed read is dropped so the next mount retries; a settings
// save drops the entry so the new mapping is read.
const configReads = new Map<string, Promise<PlanFilesConfig | null>>();

function readConfig(workspaceId: string): Promise<PlanFilesConfig | null> {
  let read = configReads.get(workspaceId);
  if (!read) {
    read = getPlanFilesConfig({ workspaceId });
    configReads.set(workspaceId, read);
    read.catch(() => configReads.delete(workspaceId));
  }
  return read;
}

/** Forgets the cached plan-files config of a workspace. */
export function invalidatePlanBoardConfig(workspaceId: string): void {
  configReads.delete(workspaceId);
}

export type PlanBoard = {
  config: PlanFilesConfig | null;
  /** True when the workflow is the workspace's plan board and sync is enabled. */
  isPlanBoard: (workflowId: string | null | undefined) => boolean;
  /** The step a board status is mapped to, or null when unmapped. */
  stepFor: (status: PlanFileStatus) => string | null;
};

/**
 * The workspace's plan-files config, read once per workspace while the
 * planFiles feature is on. Without the feature, a config, or a workspace, no
 * workflow is a plan board.
 */
export function usePlanBoard(workspaceId: string | null | undefined): PlanBoard {
  const enabled = useFeature("planFiles");
  const [loaded, setLoaded] = useState<{
    workspaceId: string;
    config: PlanFilesConfig | null;
  } | null>(null);

  useEffect(() => {
    if (!enabled || !workspaceId) return;
    let cancelled = false;
    readConfig(workspaceId)
      .then((config) => {
        if (!cancelled) setLoaded({ workspaceId, config });
      })
      .catch(() => {
        if (!cancelled) setLoaded({ workspaceId, config: null });
      });
    return () => {
      cancelled = true;
    };
  }, [enabled, workspaceId]);

  const config =
    enabled && workspaceId && loaded?.workspaceId === workspaceId && loaded.config?.enabled
      ? loaded.config
      : null;
  const isPlanBoard = useCallback(
    (workflowId: string | null | undefined) =>
      config !== null && Boolean(workflowId) && config.workflow_id === workflowId,
    [config],
  );
  const stepFor = useCallback(
    (status: PlanFileStatus) => config?.status_steps[status] ?? null,
    [config],
  );
  return { config, isPlanBoard, stepFor };
}
