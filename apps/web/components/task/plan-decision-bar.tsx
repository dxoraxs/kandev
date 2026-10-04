"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconChevronDown, IconInfoCircle } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@kandev/ui/dropdown-menu";
import { usePlanBoard } from "@/hooks/domains/plans/use-plan-board";
import { usePlanDecision, type PlanDecisionState } from "@/hooks/domains/plans/use-plan-decision";
import { PlanDecisionDialog } from "./plan-decision-dialog";
import { PlanDecisionDrawer } from "./plan-decision-drawer";
import type { PlanDecisionIntent } from "./plan-decision-intent";

export type PlanDecisionBarProps = {
  taskId: string;
  workspaceId: string | null | undefined;
  workflowId: string | null | undefined;
  workflowStepId: string | null | undefined;
  /** Desktop is a slim bar with dialogs; phone is a full-width trigger with a drawer. */
  presentation: "desktop" | "phone";
};

/**
 * The owner's decision on a plan that waits for them. Rendered only for a task
 * of the plan board that sits in the step mapped to waiting_owner, and only
 * while the planFiles feature is on.
 */
export function PlanDecisionBar({
  taskId,
  workspaceId,
  workflowId,
  workflowStepId,
  presentation,
}: PlanDecisionBarProps) {
  const board = usePlanBoard(workspaceId);
  const decision = usePlanDecision(taskId, workflowStepId ?? null);
  const waitingStep = board.stepFor("waiting_owner");
  const applies =
    board.isPlanBoard(workflowId) && waitingStep !== null && workflowStepId === waitingStep;
  if (!applies || decision.hidden) return null;
  if (presentation === "phone") {
    return (
      <div className="px-2 pb-1" data-testid="plan-decision-bar">
        <PlanDecisionDrawer decision={decision} />
      </div>
    );
  }
  return <PlanDecisionDesktopBar decision={decision} />;
}

function PlanDecisionDesktopBar({ decision }: { decision: PlanDecisionState }) {
  const { t } = useTranslation();
  const [intent, setIntent] = useState<PlanDecisionIntent | null>(null);
  const open = (next: PlanDecisionIntent) => {
    decision.clearError();
    setIntent(next);
  };
  return (
    <div
      className="flex shrink-0 items-center gap-3 border-b bg-muted/40 px-4 py-1.5 text-xs"
      data-testid="plan-decision-bar"
    >
      <IconInfoCircle className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
      <span className="min-w-0 flex-1 truncate">{t("planFiles:decisionWaiting")}</span>
      <Button
        variant="outline"
        className="cursor-pointer"
        onClick={() => open("return")}
        data-testid="plan-decision-return"
      >
        {t("planFiles:decisionReturn")}
      </Button>
      <div className="flex">
        <Button
          className="cursor-pointer rounded-r-none"
          onClick={() => open("accept")}
          data-testid="plan-decision-accept"
        >
          {t("planFiles:decisionAccept")}
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="icon"
              className="cursor-pointer rounded-l-none border-l-primary-foreground/30"
              aria-label={t("planFiles:decisionAcceptOptions")}
              data-testid="plan-decision-accept-menu"
            >
              <IconChevronDown aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              className="cursor-pointer"
              onSelect={() => open("acceptQueue")}
              data-testid="plan-decision-accept-queue"
            >
              {t("planFiles:decisionAcceptQueue")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <PlanDecisionDialog intent={intent} onClose={() => setIntent(null)} decision={decision} />
    </div>
  );
}
