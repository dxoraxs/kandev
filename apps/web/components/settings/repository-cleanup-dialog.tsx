"use client";

import { useTranslation } from "react-i18next";
import { IconLoader2 } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@kandev/ui/dialog";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@kandev/ui/drawer";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import { useMaintenanceTaskStart } from "@/hooks/domains/settings/use-maintenance-task-start";
import { settingsActionClassName } from "./settings-control";

type CleanupDialogProps = {
  repositoryId: string;
  repositoryName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

type ActionsProps = {
  pending: boolean;
  stacked: boolean;
  onCancel: () => void;
  onStart: () => void;
};

const STEP_KEYS = [
  "workspaces:cleanupStepCommit",
  "workspaces:cleanupStepMerge",
  "workspaces:cleanupStepConfirm",
  "workspaces:cleanupStepFinish",
] as const;

function CleanupActions({ pending, stacked, onCancel, onStart }: ActionsProps) {
  const { t } = useTranslation();
  const width = stacked ? "w-full" : "w-auto";
  const cancel = (
    <Button
      key="cancel"
      type="button"
      variant="outline"
      disabled={pending}
      onClick={onCancel}
      className={settingsActionClassName(`${width} cursor-pointer`)}
      data-testid="repository-cleanup-cancel"
    >
      {t("workspaces:cleanupCancel")}
    </Button>
  );
  const start = (
    <Button
      key="start"
      type="button"
      disabled={pending}
      onClick={onStart}
      className={settingsActionClassName(`${width} cursor-pointer`)}
      data-testid="repository-cleanup-confirm"
    >
      {pending && <IconLoader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />}
      {t("workspaces:cleanupStart")}
    </Button>
  );
  // The primary action comes first in the phone column and last in the desktop row.
  return stacked ? [start, cancel] : [cancel, start];
}

function CleanupSteps({ pending }: { pending: boolean }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-3 text-sm">
      <ol className="list-decimal space-y-1.5 pl-5">
        {STEP_KEYS.map((key) => (
          <li key={key}>{t(key)}</li>
        ))}
      </ol>
      <p className="text-xs text-muted-foreground">{t("workspaces:cleanupProtectedNote")}</p>
      <div role="status" className="sr-only">
        {pending ? t("workspaces:cleanupStarting") : ""}
      </div>
    </div>
  );
}

/**
 * Confirmation before a repository cleanup task starts: a centered dialog on
 * wider screens, an inset bottom drawer on phones. It stays open when the
 * launcher rejects so the reason toast and the dialog share one context.
 */
export function RepositoryCleanupDialog({
  repositoryId,
  repositoryName,
  open,
  onOpenChange,
}: CleanupDialogProps) {
  const { t } = useTranslation();
  const { isMobile } = useResponsiveBreakpoint();
  const { start, pendingRepositoryId } = useMaintenanceTaskStart();
  const pending = pendingRepositoryId !== null;

  const startCleanup = async () => {
    if (await start(repositoryId, "repository_cleanup")) onOpenChange(false);
  };
  const handleOpenChange = (next: boolean) => {
    if (!next && !pending) onOpenChange(false);
  };
  const actions = (
    <CleanupActions
      pending={pending}
      stacked={isMobile}
      onCancel={() => onOpenChange(false)}
      onStart={() => void startCleanup()}
    />
  );
  const title = t("workspaces:cleanupTitle", { name: repositoryName });
  const intro = t("workspaces:cleanupIntro");

  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={handleOpenChange}>
        <DrawerContent data-testid="repository-cleanup-dialog" className="max-h-[85dvh]">
          <DrawerHeader className="text-left">
            <DrawerTitle>{title}</DrawerTitle>
            <DrawerDescription>{intro}</DrawerDescription>
          </DrawerHeader>
          <div className="overflow-y-auto px-4">
            <CleanupSteps pending={pending} />
          </div>
          <DrawerFooter className="pb-[calc(1rem+env(safe-area-inset-bottom))]">
            {actions}
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
    );
  }
  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent data-testid="repository-cleanup-dialog">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{intro}</DialogDescription>
        </DialogHeader>
        <CleanupSteps pending={pending} />
        <DialogFooter>{actions}</DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
