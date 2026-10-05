import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  commitPlanFiles,
  getPlanGitStatus,
  planCommitError,
  type PlanCommitErrorCode,
  type PlanGitRepository,
} from "@/lib/api/domains/plan-files-api";

export type PlanCommitFailure = {
  /** Localized reason. */
  message: string;
  /** The last lines of git output, empty when git printed nothing relevant. */
  output: string;
};

const FAILURE_KEY: Partial<Record<PlanCommitErrorCode, string>> = {
  repository_busy: "planFiles:gitCommitBusy",
  nothing_to_commit: "planFiles:gitCommitNothing",
  commit_failed: "planFiles:gitCommitFailed",
};

/**
 * Plan files with uncommitted changes per local repository of the workspace,
 * and the commit action. A failed read lists nothing: the block is an offer,
 * never a blocker. `refreshKey` re-reads, for example after a sync pass.
 */
export function usePlanGitStatus(workspaceId: string, refreshKey?: string) {
  const { t } = useTranslation();
  const [repositories, setRepositories] = useState<PlanGitRepository[]>([]);
  const [reloads, setReloads] = useState(0);
  const [committing, setCommitting] = useState(false);
  const [failure, setFailure] = useState<PlanCommitFailure | null>(null);

  useEffect(() => {
    let cancelled = false;
    getPlanGitStatus({ workspaceId })
      .then((next) => {
        if (!cancelled) setRepositories(next);
      })
      .catch(() => {
        if (!cancelled) setRepositories([]);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId, refreshKey, reloads]);

  const clearFailure = useCallback(() => setFailure(null), []);

  /** Resolves true when the commit was created. */
  const commit = useCallback(
    async (repositoryId: string, message: string) => {
      setCommitting(true);
      setFailure(null);
      try {
        await commitPlanFiles({ repository_id: repositoryId, message }, { workspaceId });
        setReloads((n) => n + 1);
        return true;
      } catch (err) {
        const rejected = planCommitError(err);
        const key = (rejected && FAILURE_KEY[rejected.code]) ?? "planFiles:gitCommitError";
        setFailure({ message: t(key), output: rejected?.output ?? "" });
        if (rejected?.code === "nothing_to_commit") setReloads((n) => n + 1);
        return false;
      } finally {
        setCommitting(false);
      }
    },
    [workspaceId, t],
  );

  return { repositories, committing, failure, clearFailure, commit };
}

export type PlanGitStatusState = ReturnType<typeof usePlanGitStatus>;
