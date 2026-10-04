"use client";

import { useTranslation } from "react-i18next";
import { IconAlertTriangle, IconCheck, IconClock } from "@tabler/icons-react";
import { formatDateTime } from "@/lib/i18n/formats";
import type { PlanFilesConfig } from "@/lib/api/domains/plan-files-api";

// Closed set of failure reasons the backend reports; an unknown value is shown
// as received.
const REASON_KEYS: Record<string, string> = {
  parse: "planFiles:reasonParse",
  duplicate_external_id: "planFiles:reasonDuplicateExternalId",
  task_service: "planFiles:reasonTaskService",
  workflow_missing: "planFiles:reasonWorkflowMissing",
  repository_list: "planFiles:reasonRepositoryList",
  root_missing: "planFiles:reasonRootMissing",
  outside_root: "planFiles:reasonOutsideRoot",
  truncated: "planFiles:reasonTruncated",
  not_regular_file: "planFiles:reasonNotRegularFile",
  too_large: "planFiles:reasonTooLarge",
  unreadable: "planFiles:reasonUnreadable",
  write_failed: "planFiles:reasonWriteFailed",
};

function Outcome({ ok, failed }: { ok: boolean; failed: number }) {
  const { t } = useTranslation();
  if (ok && failed === 0) {
    return (
      <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
        <IconCheck className="h-4 w-4" aria-hidden="true" />
        {t("planFiles:outcomeOk")}
      </span>
    );
  }
  return (
    <span className="flex items-center gap-1 text-destructive">
      <IconAlertTriangle className="h-4 w-4" aria-hidden="true" />
      {ok ? t("planFiles:outcomePartial") : t("planFiles:outcomeFailed")}
    </span>
  );
}

/** Time and outcome of the last pass, its counts, and one row per failed file. */
export function PlanFilesStatus({ config }: { config: PlanFilesConfig }) {
  const { t } = useTranslation();
  if (!config.last_pass_at) {
    return (
      <p
        className="flex items-center gap-1.5 text-xs text-muted-foreground"
        data-testid="plan-files-status"
        data-state="waiting"
      >
        <IconClock className="h-4 w-4" aria-hidden="true" />
        {t("planFiles:neverSynced")}
      </p>
    );
  }
  const counts = config.last_counts;
  const errors = config.last_file_errors ?? [];
  return (
    <div
      className="space-y-2 text-xs"
      data-testid="plan-files-status"
      data-state={config.last_pass_ok ? "ok" : "failed"}
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="text-muted-foreground">
          {t("planFiles:lastSync", { when: formatDateTime(config.last_pass_at) })}
        </span>
        <Outcome ok={config.last_pass_ok} failed={counts.failed} />
      </div>
      <p className="text-muted-foreground" data-testid="plan-files-counts">
        {[
          t("planFiles:countCreated", { count: counts.created }),
          t("planFiles:countUpdated", { count: counts.updated }),
          t("planFiles:countMoved", { count: counts.moved }),
          t("planFiles:countArchived", { count: counts.archived }),
          t("planFiles:countFailed", { count: counts.failed }),
        ].join(" · ")}
      </p>
      {errors.length > 0 && (
        <ul className="space-y-1" data-testid="plan-files-file-errors">
          {errors.map((row, index) => (
            <li
              key={`${row.repository_id}:${row.rel_path}:${index}`}
              className="flex flex-col gap-0.5 text-destructive sm:flex-row sm:gap-2"
            >
              <span className="flex min-w-0 items-start gap-1">
                <IconAlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                <span className="min-w-0 break-all font-mono">
                  {row.repository_name ? `${row.repository_name}: ` : ""}
                  {row.rel_path}
                </span>
              </span>
              <span>{REASON_KEYS[row.reason] ? t(REASON_KEYS[row.reason]!) : row.reason}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
