"use client";

import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@kandev/ui/tooltip";
import { BroomIcon } from "@/components/icons/broom-icon";
import { RepositoryCleanupDialog } from "@/components/settings/repository-cleanup-dialog";
import { useToast } from "@/components/toast-provider";
import { useFeature } from "@/hooks/domains/features/use-feature";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import { resolveCleanupTarget } from "@/lib/kanban/cleanup-target";
import type { TaskListingPage } from "@/lib/task-listing/view-navigation";
import type { Repository } from "@/lib/types/http";

/**
 * The board's cleanup action. `onOpen` is undefined while the owner still has
 * to choose a repository in the board filter.
 */
export type BoardCleanupAction = {
  onOpen: (() => void) | undefined;
  dialog: ReactNode;
};

/**
 * The cleanup action of the kanban top bar, or null when it is not offered:
 * the flag is off, the page is not the kanban board, or the workspace has no
 * local repository.
 */
export function useBoardCleanupAction({
  currentPage,
  repositories,
  selectedRepositoryId,
}: {
  currentPage: TaskListingPage;
  repositories: Repository[];
  selectedRepositoryId: string | null;
}): BoardCleanupAction | null {
  const enabled = useFeature("repositoryCleanup");
  const [open, setOpen] = useState(false);
  const target = resolveCleanupTarget(repositories, selectedRepositoryId);
  if (!enabled || currentPage !== "kanban" || target.kind === "hidden") return null;
  if (target.kind === "needs_selection") return { onOpen: undefined, dialog: null };
  const repository = target.repository;
  return {
    onOpen: () => setOpen(true),
    dialog: (
      <RepositoryCleanupDialog
        repositoryId={repository.id}
        repositoryName={repository.name ?? ""}
        open={open}
        onOpenChange={setOpen}
      />
    ),
  };
}

/** Icon-only broom of the desktop and tablet top bar. */
export function BoardCleanupButton({ action }: { action: BoardCleanupAction }) {
  const { t } = useTranslation();
  const { toast } = useToast();
  const { isFinePointer } = useResponsiveBreakpoint();
  const ready = action.onOpen !== undefined;
  const label = t("workspaces:cleanupRepository");
  const hint = t("workspaces:cleanupChooseRepository");
  // aria-disabled instead of disabled: a disabled button receives no pointer
  // events, so its tooltip could never explain why it is unavailable.
  const button = (
    <Button
      type="button"
      variant="outline"
      size="icon-lg"
      className={ready ? "cursor-pointer" : "cursor-not-allowed opacity-50"}
      aria-label={label}
      aria-disabled={ready ? undefined : true}
      data-testid="board-cleanup-button"
      onClick={ready ? action.onOpen : () => toast({ description: hint, variant: "default" })}
    >
      <BroomIcon className="h-4 w-4" />
    </Button>
  );
  if (!isFinePointer) return button;
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{button}</TooltipTrigger>
        <TooltipContent>{ready ? label : hint}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

/** Full-width broom entry of the phone listing menu. */
export function MobileCleanupEntry({ onOpen }: { onOpen: (() => void) | undefined }) {
  const { t } = useTranslation();
  const ready = onOpen !== undefined;
  return (
    <div className="space-y-1">
      <Button
        variant="outline"
        className="min-h-11 w-full cursor-pointer justify-start gap-2"
        onClick={onOpen}
        disabled={!ready}
        aria-describedby={ready ? undefined : "mobile-cleanup-hint"}
        data-testid="mobile-display-cleanup"
      >
        <BroomIcon className="h-4 w-4" />
        {t("workspaces:cleanupRepository")}
      </Button>
      {!ready && (
        <p id="mobile-cleanup-hint" className="text-xs text-muted-foreground">
          {t("workspaces:cleanupChooseRepository")}
        </p>
      )}
    </div>
  );
}
