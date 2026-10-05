"use client";

import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { IconFilePlus } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { NewPlanDialog } from "./new-plan-dialog";
import { usePlanBoard } from "@/hooks/domains/plans/use-plan-board";
import type { TaskListingPage } from "@/lib/task-listing/view-navigation";

/**
 * Whether the New plan action is offered and whether its form is open. The
 * action exists only on the kanban page while the active workflow is the
 * workspace's plan board.
 */
export function useNewPlanAction(
  workspaceId: string | null | undefined,
  currentPage: TaskListingPage,
  workflowId: string | null | undefined,
) {
  const board = usePlanBoard(workspaceId);
  const [requested, setRequested] = useState(false);
  const available =
    Boolean(workspaceId) && currentPage === "kanban" && board.isPlanBoard(workflowId);
  return { available, open: available && requested, setOpen: setRequested };
}

/** Header button of the New plan action; `compact` is the icon-only tablet form. */
export function NewPlanButton({ onClick, compact }: { onClick: () => void; compact?: boolean }) {
  const { t } = useTranslation();
  const label = t("planFiles:newPlanTitle");
  return (
    <Button
      variant="outline"
      size={compact ? "icon-lg" : "lg"}
      className="cursor-pointer"
      onClick={onClick}
      data-testid="new-plan-button"
      data-compact={compact ? "true" : undefined}
    >
      <IconFilePlus className="h-4 w-4" aria-hidden />
      {compact ? <span className="sr-only">{label}</span> : label}
    </Button>
  );
}

/**
 * The New plan action of a board header: the callback that opens the form, or
 * undefined off the plan board, and the form itself to render once.
 */
export function useNewPlanEntry(
  workspaceId: string | null | undefined,
  currentPage: TaskListingPage,
  workflowId: string | null | undefined,
): { onNewPlan: (() => void) | undefined; dialog: ReactNode } {
  const action = useNewPlanAction(workspaceId, currentPage, workflowId);
  if (!action.available || !workspaceId) return { onNewPlan: undefined, dialog: null };
  return {
    onNewPlan: () => action.setOpen(true),
    dialog: (
      <NewPlanDialog workspaceId={workspaceId} open={action.open} onOpenChange={action.setOpen} />
    ),
  };
}
