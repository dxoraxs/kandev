"use client";

import { useTranslation } from "react-i18next";
import { IconLoader2 } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { useMaintenanceTaskStart } from "@/hooks/domains/settings/use-maintenance-task-start";
import { useUnadaptedPlanFiles } from "@/hooks/domains/settings/use-unadapted-plan-files";
import { settingsActionClassName } from "./settings-control";

type UnadaptedRowsProps = {
  workspaceId: string;
  /** True when no plan-files config exists, so starting adaptation creates the board. */
  boardWillBeCreated: boolean;
  refreshKey?: string;
};

/**
 * One row per local repository holding plan files without board status, each
 * with the Adapt with agent action. Renders nothing when there are none.
 */
export function PlanFilesUnadaptedRows({
  workspaceId,
  boardWillBeCreated,
  refreshKey,
}: UnadaptedRowsProps) {
  const { t } = useTranslation();
  const rows = useUnadaptedPlanFiles(workspaceId, refreshKey);
  const { start, pendingRepositoryId } = useMaintenanceTaskStart();
  if (rows.length === 0) return null;
  const busy = pendingRepositoryId !== null;

  return (
    <div className="space-y-2" data-testid="plan-files-unadapted">
      <h4 className="text-xs font-medium">{t("planFiles:unadaptedTitle")}</h4>
      <ul className="space-y-2">
        {rows.map((row) => (
          <li
            key={row.repository_id}
            className="flex flex-col gap-2 rounded-md border p-3 md:flex-row md:items-center md:justify-between"
            data-testid="plan-files-unadapted-row"
            data-repository-id={row.repository_id}
          >
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{row.repository_name}</p>
              <p className="text-xs text-muted-foreground" data-testid="plan-files-unadapted-count">
                {t("planFiles:unadaptedCount", { count: row.count })}
              </p>
              {boardWillBeCreated && (
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("planFiles:boardWillBeCreated")}
                </p>
              )}
            </div>
            <Button
              type="button"
              size="sm"
              disabled={busy}
              onClick={() => void start(row.repository_id, "plan_adaptation")}
              className={settingsActionClassName("w-full shrink-0 cursor-pointer md:w-auto")}
              data-testid="plan-files-adapt"
            >
              {pendingRepositoryId === row.repository_id && (
                <IconLoader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
              )}
              {t("planFiles:adaptWithAgent")}
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}
