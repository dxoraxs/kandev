import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "@/lib/toast/sonner";
import {
  createPlan,
  planCreateErrorCode,
  type CreatePlanBody,
  type PlanCreateErrorCode,
} from "@/lib/api/domains/plan-files-api";

export type CreatePlanState = {
  pending: boolean;
  /** Localized message of the last failed create, or null. */
  error: string | null;
  clearError: () => void;
  /** Resolves true when the plan file and its task exist. */
  submit: (body: CreatePlanBody) => Promise<boolean>;
};

const FAILURE_KEY: Record<PlanCreateErrorCode, string> = {
  file_exists: "planFiles:newPlanFileExists",
  invalid_plan: "planFiles:newPlanInvalid",
  repository_not_found: "planFiles:newPlanRepositoryGone",
};

/** Submit state of the New plan form for one workspace. */
export function useCreatePlan(workspaceId: string): CreatePlanState {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const clearError = useCallback(() => setError(null), []);

  const submit = useCallback(
    async (body: CreatePlanBody) => {
      setPending(true);
      setError(null);
      try {
        await createPlan(body, { workspaceId });
        toast.success(t("planFiles:newPlanCreated"));
        return true;
      } catch (err) {
        const code = planCreateErrorCode(err);
        setError(t(code ? FAILURE_KEY[code] : "planFiles:newPlanFailed"));
        return false;
      } finally {
        setPending(false);
      }
    },
    [workspaceId, t],
  );

  return { pending, error, clearError, submit };
}
