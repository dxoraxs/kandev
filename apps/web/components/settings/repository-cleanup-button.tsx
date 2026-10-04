"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@kandev/ui/tooltip";
import { BroomIcon } from "@/components/icons/broom-icon";
import { useFeature } from "@/hooks/domains/features/use-feature";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import { RepositoryCleanupDialog } from "./repository-cleanup-dialog";

type CleanupButtonProps = {
  repository: { id: string; name?: string | null; source_type?: string | null };
  /** Read-only repositories (the dedicated Improve Kandev workspace) offer no actions. */
  readOnly?: boolean;
};

/**
 * Icon-only broom action on a local repository row. It opens the cleanup
 * confirmation; nothing starts until the owner confirms.
 */
export function RepositoryCleanupButton({ repository, readOnly = false }: CleanupButtonProps) {
  const { t } = useTranslation();
  const enabled = useFeature("repositoryCleanup");
  const { isFinePointer } = useResponsiveBreakpoint();
  const [open, setOpen] = useState(false);
  if (!enabled || readOnly || repository.source_type !== "local") return null;

  const label = t("workspaces:cleanupRepository");
  const button = (
    <Button
      type="button"
      variant="outline"
      size="icon"
      className="cursor-pointer"
      aria-label={label}
      data-testid="repository-cleanup-button"
      onClick={() => setOpen(true)}
    >
      <BroomIcon className="h-4 w-4" />
    </Button>
  );

  // The dialog is a React child of the clickable card, so its events would bubble
  // to the card's open-editor handler through the portal.
  return (
    <span className="contents" onClick={(event) => event.stopPropagation()}>
      {isFinePointer ? (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>{button}</TooltipTrigger>
            <TooltipContent>{label}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ) : (
        button
      )}
      <RepositoryCleanupDialog
        repositoryId={repository.id}
        repositoryName={repository.name ?? ""}
        open={open}
        onOpenChange={setOpen}
      />
    </span>
  );
}
