"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@kandev/ui/dialog";
import { Drawer, DrawerContent, DrawerHeader, DrawerTitle } from "@kandev/ui/drawer";
import { Input } from "@kandev/ui/input";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import type { PlanCommitFailure } from "@/hooks/domains/plans/use-plan-git-status";
import {
  DEFAULT_PLAN_COMMIT_MESSAGE,
  type PlanGitRepository,
} from "@/lib/api/domains/plan-files-api";
import { settingsControlClassName } from "./settings-control";
import { SettingsFieldLabel } from "./settings-typography";

export type PlanFilesCommitProps = {
  /** The repository being committed, or null while the surface is closed. */
  repository: PlanGitRepository | null;
  committing: boolean;
  failure: PlanCommitFailure | null;
  onCommit: (repositoryId: string, message: string) => Promise<boolean>;
  onClose: () => void;
};

/**
 * Confirmation of one plan files commit: the files that go into it and an
 * editable message. A dialog on desktop, a bottom drawer on phones.
 */
export function PlanFilesCommit(props: PlanFilesCommitProps) {
  const { t } = useTranslation();
  const { isMobile } = useResponsiveBreakpoint();
  const { repository, onClose } = props;
  const title = t("planFiles:gitCommitTitle", { name: repository?.repository_name ?? "" });
  const onOpenChange = (open: boolean) => {
    if (!open) onClose();
  };
  if (isMobile) {
    return (
      <Drawer open={repository !== null} onOpenChange={onOpenChange}>
        <DrawerContent data-testid="plan-files-commit-drawer">
          <DrawerHeader className="text-left">
            <DrawerTitle>{title}</DrawerTitle>
          </DrawerHeader>
          <div
            className="px-4"
            style={{ paddingBottom: "calc(1rem + env(safe-area-inset-bottom, 0px))" }}
          >
            {repository ? <CommitForm {...props} repository={repository} touch /> : null}
          </div>
        </DrawerContent>
      </Drawer>
    );
  }
  return (
    <Dialog open={repository !== null} onOpenChange={onOpenChange}>
      {repository ? (
        <DialogContent enterConfirms={false} data-testid="plan-files-commit-dialog">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
          <CommitForm {...props} repository={repository} touch={false} />
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

type CommitFormProps = Omit<PlanFilesCommitProps, "repository"> & {
  repository: PlanGitRepository;
  touch: boolean;
};

function CommitForm({
  repository,
  committing,
  failure,
  onCommit,
  onClose,
  touch,
}: CommitFormProps) {
  const { t } = useTranslation();
  const [message, setMessage] = useState(DEFAULT_PLAN_COMMIT_MESSAGE);
  const confirm = async () => {
    if (await onCommit(repository.repository_id, message.trim())) onClose();
  };
  const buttonClass = touch ? "min-h-11 w-full cursor-pointer" : "cursor-pointer";
  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t("planFiles:gitCommitHint")}</p>
      <ul
        aria-label={t("planFiles:gitCommitFiles")}
        className="max-h-40 space-y-0.5 overflow-auto rounded-md border bg-muted/30 p-2 font-mono text-xs"
        data-testid="plan-files-commit-files"
      >
        {repository.files.map((file) => (
          <li key={file} className="break-all">
            {file}
          </li>
        ))}
      </ul>
      <div className="space-y-1.5">
        <SettingsFieldLabel htmlFor="plan-files-commit-message">
          {t("planFiles:gitCommitMessage")}
        </SettingsFieldLabel>
        <Input
          id="plan-files-commit-message"
          value={message}
          onChange={(event) => setMessage(event.target.value)}
          className={settingsControlClassName("w-full")}
          data-testid="plan-files-commit-message"
        />
      </div>
      {failure ? <CommitFailureText failure={failure} /> : null}
      <DialogFooter className={touch ? "flex-col gap-2 sm:flex-col" : undefined}>
        <Button
          variant="outline"
          className={buttonClass}
          onClick={onClose}
          data-testid="plan-files-commit-cancel"
        >
          {t("planFiles:gitCommitCancel")}
        </Button>
        <Button
          className={buttonClass}
          disabled={committing}
          onClick={confirm}
          data-testid="plan-files-commit-confirm"
        >
          {t("planFiles:gitCommitConfirm")}
        </Button>
      </DialogFooter>
    </div>
  );
}

function CommitFailureText({ failure }: { failure: PlanCommitFailure }) {
  const { t } = useTranslation();
  return (
    <div role="alert" className="space-y-1 text-xs" data-testid="plan-files-commit-error">
      <p className="text-destructive">{failure.message}</p>
      {failure.output ? (
        <pre
          aria-label={t("planFiles:gitCommitOutput")}
          className="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-muted/30 p-2 font-mono"
          data-testid="plan-files-commit-output"
        >
          {failure.output}
        </pre>
      ) : null}
    </div>
  );
}
