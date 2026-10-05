import { useEffect, useState } from "react";
import {
  getUnadaptedPlanFiles,
  type UnadaptedPlanFilesRepository,
} from "@/lib/api/domains/plan-files-api";

/**
 * Repositories with plan files that lack board status. A failed read lists
 * nothing: the rows are an offer, never a blocker. `refreshKey` re-reads, for
 * example after a sync pass.
 */
export function useUnadaptedPlanFiles(workspaceId: string, refreshKey?: string) {
  const [rows, setRows] = useState<UnadaptedPlanFilesRepository[]>([]);
  useEffect(() => {
    let cancelled = false;
    getUnadaptedPlanFiles({ workspaceId })
      .then((next) => {
        if (!cancelled) setRows(next);
      })
      .catch(() => {
        if (!cancelled) setRows([]);
      });
    return () => {
      cancelled = true;
    };
  }, [workspaceId, refreshKey]);
  return rows;
}
