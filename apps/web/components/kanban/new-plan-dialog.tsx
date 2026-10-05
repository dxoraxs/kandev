"use client";

import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@kandev/ui/dialog";
import { Drawer, DrawerContent, DrawerHeader, DrawerTitle } from "@kandev/ui/drawer";
import { useCreatePlan } from "@/hooks/domains/plans/use-create-plan";
import { usePlanBoard } from "@/hooks/domains/plans/use-plan-board";
import { useRepositories } from "@/hooks/domains/workspace/use-repositories";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import { NewPlanForm } from "./new-plan-form";

const LOCAL_SOURCE = "local";

export type NewPlanDialogProps = {
  workspaceId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * New plan: asks for a local repository, one of the scanned directories, a
 * title, and the optional file name, priority, executor, and body. A dialog on
 * desktop and tablet, a full-height drawer on phones.
 */
export function NewPlanDialog({ workspaceId, open, onOpenChange }: NewPlanDialogProps) {
  const { t } = useTranslation();
  const { isMobile } = useResponsiveBreakpoint();
  const { config } = usePlanBoard(workspaceId);
  const { repositories } = useRepositories(workspaceId, open);
  const create = useCreatePlan(workspaceId);
  const local = useMemo(
    () =>
      repositories
        .filter((repo) => repo.source_type === LOCAL_SOURCE && repo.local_path !== "")
        .sort((a, b) => a.name.localeCompare(b.name)),
    [repositories],
  );
  const executors = useMemo(
    () => Array.from(new Set(Object.values(config?.executor_steps ?? {}))),
    [config],
  );
  const handleOpenChange = (next: boolean) => {
    if (!next) create.clearError();
    onOpenChange(next);
  };
  const title = t("planFiles:newPlanTitle");
  const form = (touch: boolean) => (
    <NewPlanForm
      repositories={local}
      directories={config?.directories ?? []}
      executors={executors}
      create={create}
      onClose={() => handleOpenChange(false)}
      touch={touch}
    />
  );
  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={handleOpenChange}>
        <DrawerContent
          className="h-[calc(100dvh-16px-env(safe-area-inset-bottom,0px))] !max-h-[calc(100dvh-16px-env(safe-area-inset-bottom,0px))] outline-none"
          data-testid="new-plan-drawer"
        >
          <DrawerHeader className="shrink-0 text-left">
            <DrawerTitle>{title}</DrawerTitle>
          </DrawerHeader>
          {open ? form(true) : null}
        </DrawerContent>
      </Drawer>
    );
  }
  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent enterConfirms={false} data-testid="new-plan-dialog">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        {open ? form(false) : null}
      </DialogContent>
    </Dialog>
  );
}
