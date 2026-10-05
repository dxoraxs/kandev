import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useToast } from "@/components/toast-provider";
import { useRouter } from "@/lib/routing/client-router";
import { linkToTask } from "@/lib/links";
import {
  MaintenanceTaskError,
  startRepositoryMaintenanceTask,
  type MaintenanceFailureReason,
  type RepositoryMaintenanceKind,
} from "@/lib/api/domains/repository-maintenance-api";

const REASON_KEYS: Record<MaintenanceFailureReason, string> = {
  no_agent_profile: "workspaces:maintenanceNoAgentProfile",
  no_workflow: "workspaces:maintenanceNoWorkflow",
  repository_not_local: "workspaces:maintenanceRepositoryNotLocal",
  kind_unavailable: "workspaces:maintenanceKindUnavailable",
  repository_not_found: "workspaces:maintenanceRepositoryNotFound",
  failed: "workspaces:maintenanceFailed",
};

/**
 * Starts a repository maintenance task and opens it. An existing active task is
 * opened instead (with an info toast); an empty session id is a normal state of
 * such a task. Failures surface as a localized toast, and a task created before
 * a launch failure is still opened so the owner can retry from it.
 */
export function useMaintenanceTaskStart() {
  const { t } = useTranslation();
  const { toast } = useToast();
  const router = useRouter();
  const [pendingRepositoryId, setPendingRepositoryId] = useState<string | null>(null);
  const inFlight = useRef(false);

  const start = useCallback(
    async (repositoryId: string, kind: RepositoryMaintenanceKind): Promise<boolean> => {
      if (inFlight.current) return false;
      inFlight.current = true;
      setPendingRepositoryId(repositoryId);
      try {
        const result = await startRepositoryMaintenanceTask(repositoryId, kind);
        if (result.existing) {
          toast({ description: t("workspaces:maintenanceExisting"), variant: "default" });
        }
        router.push(linkToTask(result.task_id));
        return true;
      } catch (err) {
        const reason = err instanceof MaintenanceTaskError ? err.reason : "failed";
        toast({ description: t(REASON_KEYS[reason]), variant: "error" });
        if (err instanceof MaintenanceTaskError && err.taskId) {
          router.push(linkToTask(err.taskId));
        }
        return false;
      } finally {
        inFlight.current = false;
        setPendingRepositoryId(null);
      }
    },
    [router, t, toast],
  );

  return { start, pendingRepositoryId };
}
