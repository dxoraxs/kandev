"use client";

import { useTranslation } from "react-i18next";
import { IconLoader2, IconRefresh } from "@tabler/icons-react";
import { Alert, AlertDescription } from "@kandev/ui/alert";
import { Button } from "@kandev/ui/button";
import { Switch } from "@kandev/ui/switch";
import { useFeature } from "@/hooks/domains/features/use-feature";
import { usePlanFiles } from "@/hooks/domains/settings/use-plan-files";
import { settingsActionClassName } from "./settings-control";
import { SettingsErrorText } from "./settings-typography";
import { PlanFilesBoardField, PlanFilesMapping } from "./plan-files-mapping";
import { PlanFilesDirectories } from "./plan-files-directories";
import { PlanFilesStatus } from "./plan-files-status";

/**
 * Plan files settings on the workspace Workflows tab. Renders nothing, and
 * loads nothing, while the `planFiles` runtime flag is off.
 */
export function PlanFilesSection({ workspaceId }: { workspaceId: string }) {
  const enabled = useFeature("planFiles");
  if (!enabled) return null;
  return <PlanFilesSectionBody key={workspaceId} workspaceId={workspaceId} />;
}

function PlanFilesSectionBody({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation();
  const state = usePlanFiles(workspaceId);
  const { draft } = state;
  const incomplete = draft.enabled && draft.workflowId !== "" && !state.complete;

  return (
    <div className="mb-4 md:pl-8" data-testid="plan-files-section">
      <div className="space-y-4 rounded-lg border bg-card p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-sm font-semibold">{t("planFiles:title")}</h3>
            <p className="mt-0.5 text-xs text-muted-foreground">{t("planFiles:description")}</p>
          </div>
          <Switch
            checked={draft.enabled}
            disabled={state.loading}
            onCheckedChange={(checked) => state.patch({ enabled: checked })}
            aria-label={t("planFiles:enabled")}
            data-testid="plan-files-enabled"
          />
        </div>
        {draft.enabled && (
          <>
            <PlanFilesBoardField
              workflows={state.workflows}
              workflowId={draft.workflowId}
              creating={state.creatingBoard}
              onSelect={state.selectWorkflow}
              onCreate={state.createBoard}
            />
            {draft.workflowId !== "" && (
              <PlanFilesMapping
                steps={state.steps}
                statusSteps={draft.statusSteps}
                disabled={state.saving}
                onChange={state.setStatusStep}
              />
            )}
            <PlanFilesDirectories
              directories={draft.directories}
              onChange={(directories) => state.patch({ directories })}
            />
            {incomplete && (
              <p className="text-xs text-muted-foreground" data-testid="plan-files-incomplete">
                {t("planFiles:mappingIncomplete")}
              </p>
            )}
            {state.config && <PlanFilesStatus config={state.config} />}
          </>
        )}
        {state.syncNotice === "running" && (
          <Alert data-testid="plan-files-sync-running">
            <AlertDescription>{t("planFiles:syncRunning")}</AlertDescription>
          </Alert>
        )}
        {state.error && (
          <SettingsErrorText data-testid="plan-files-error">{state.error}</SettingsErrorText>
        )}
        <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
          {draft.enabled && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={!state.canSync}
              onClick={state.syncNow}
              className={settingsActionClassName("w-full cursor-pointer sm:w-auto")}
              data-testid="plan-files-sync-now"
            >
              {state.syncing ? (
                <IconLoader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <IconRefresh className="mr-2 h-4 w-4" />
              )}
              {t("planFiles:syncNow")}
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            disabled={!state.canSave}
            onClick={state.save}
            className={settingsActionClassName("w-full cursor-pointer sm:w-auto")}
            data-testid="plan-files-save"
          >
            {t("planFiles:save")}
          </Button>
        </div>
      </div>
    </div>
  );
}
