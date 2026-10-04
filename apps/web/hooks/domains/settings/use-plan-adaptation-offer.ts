import { useCallback, useState } from "react";
import { useFeature } from "@/hooks/domains/features/use-feature";
import { getPlanFilesConfig, getUnadaptedPlanFiles } from "@/lib/api/domains/plan-files-api";
import type { PlanAdaptationOffer } from "@/components/settings/adapt-plans-offer-dialog";

/**
 * Decides whether a just-added repository deserves the plan adaptation offer:
 * the `planFiles` flag is on and the repository holds plan files without board
 * status. The check is best effort; any failure simply shows no offer.
 */
export function usePlanAdaptationOffer(workspaceId: string | null) {
  const enabled = useFeature("planFiles");
  const [offer, setOffer] = useState<PlanAdaptationOffer | null>(null);

  const check = useCallback(
    async (repository: { id: string; name: string }) => {
      if (!enabled || !workspaceId) return;
      try {
        const rows = await getUnadaptedPlanFiles({ workspaceId, repositoryId: repository.id });
        const row = rows.find((candidate) => candidate.repository_id === repository.id);
        if (!row) return;
        const config = await getPlanFilesConfig({ workspaceId }).catch(() => undefined);
        setOffer({
          repositoryId: repository.id,
          repositoryName: row.repository_name || repository.name,
          count: row.count,
          boardWillBeCreated: config === null,
        });
      } catch {
        // No offer: adding the repository has already succeeded.
      }
    },
    [enabled, workspaceId],
  );

  const dismiss = useCallback(() => setOffer(null), []);
  return { offer, check, dismiss };
}
