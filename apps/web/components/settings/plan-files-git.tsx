"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { usePlanGitStatus } from "@/hooks/domains/plans/use-plan-git-status";
import { settingsActionClassName } from "./settings-control";
import { PlanFilesCommit } from "./plan-files-git-commit";

type GitProps = {
  workspaceId: string;
  /** Re-reads the git state when it changes, for example after a sync pass. */
  refreshKey?: string;
};

/**
 * One row per local repository with the number of plan files that have
 * uncommitted changes and, when there are any, the Commit plan files action.
 * Renders nothing while the workspace has no local repository.
 */
export function PlanFilesGit({ workspaceId, refreshKey }: GitProps) {
  const { t } = useTranslation();
  const git = usePlanGitStatus(workspaceId, refreshKey);
  const [targetId, setTargetId] = useState<string | null>(null);
  if (git.repositories.length === 0) return null;
  const target = git.repositories.find((repo) => repo.repository_id === targetId) ?? null;

  return (
    <div className="space-y-2" data-testid="plan-files-git">
      <h4 className="text-xs font-medium">{t("planFiles:gitTitle")}</h4>
      <ul className="space-y-2">
        {git.repositories.map((repo) => (
          <li
            key={repo.repository_id}
            className="flex flex-col gap-2 rounded-md border p-3 md:flex-row md:items-center md:justify-between"
            data-testid="plan-files-git-row"
            data-repository-id={repo.repository_id}
          >
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{repo.repository_name}</p>
              <p className="text-xs text-muted-foreground" data-testid="plan-files-git-count">
                {t("planFiles:gitFileCount", { count: repo.files.length })}
              </p>
            </div>
            {repo.files.length > 0 && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  git.clearFailure();
                  setTargetId(repo.repository_id);
                }}
                className={settingsActionClassName("w-full shrink-0 cursor-pointer md:w-auto")}
                data-testid="plan-files-git-commit"
              >
                {t("planFiles:gitCommitAction")}
              </Button>
            )}
          </li>
        ))}
      </ul>
      <PlanFilesCommit
        repository={target}
        committing={git.committing}
        failure={git.failure}
        onCommit={git.commit}
        onClose={() => setTargetId(null)}
      />
    </div>
  );
}
