import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  decidePlan,
  planDecisionErrorCode,
  type PlanDecisionBody,
} from "@/lib/api/domains/plan-files-api";

export type PlanDecisionState = {
  pending: boolean;
  /** Localized message of the last failed decision, or null. */
  error: string | null;
  /** True once the decision no longer applies: it was recorded or the task is no plan task. */
  hidden: boolean;
  clearError: () => void;
  /** Resolves true when the decision was recorded. */
  submit: (body: PlanDecisionBody) => Promise<boolean>;
};

/**
 * Submit state of the owner decision on one plan task. The hidden flag belongs
 * to the step the decision was made in, so a plan that waits for the owner
 * again later shows the bar again.
 */
export function usePlanDecision(taskId: string, stepId: string | null): PlanDecisionState {
  const { t } = useTranslation();
  const scope = `${taskId}:${stepId ?? ""}`;
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hiddenScope, setHiddenScope] = useState<string | null>(null);
  const clearError = useCallback(() => setError(null), []);

  const submit = useCallback(
    async (body: PlanDecisionBody) => {
      setPending(true);
      setError(null);
      try {
        await decidePlan(taskId, body);
        setHiddenScope(scope);
        return true;
      } catch (err) {
        const code = planDecisionErrorCode(err);
        if (code === "not_plan_task") setHiddenScope(scope);
        else if (code === "file_changed") setError(t("planFiles:decisionFileChanged"));
        else if (code === "not_waiting_owner") setError(t("planFiles:decisionNotWaiting"));
        else setError(t("planFiles:decisionFailed"));
        return false;
      } finally {
        setPending(false);
      }
    },
    [taskId, scope, t],
  );

  return { pending, error, hidden: hiddenScope === scope, clearError, submit };
}
