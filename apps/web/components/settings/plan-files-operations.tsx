"use client";

import { useTranslation } from "react-i18next";
import { IconPlus, IconX } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { Checkbox } from "@kandev/ui/checkbox";
import { Input } from "@kandev/ui/input";
import { settingsActionClassName, settingsControlClassName } from "./settings-control";
import { SettingsFieldLabel } from "./settings-typography";
import { NativeSelect } from "./plan-files-mapping";
import type {
  ExecutorRow,
  PlanFilesDraft,
  PlanFilesStep,
} from "@/hooks/domains/settings/use-plan-files";

const FIELD_ROW = "grid gap-1.5 sm:grid-cols-[14rem_minmax(0,1fr)] sm:items-center sm:gap-3";

type OperationsProps = {
  draft: PlanFilesDraft;
  steps: PlanFilesStep[];
  disabled: boolean;
  onChange: (changes: Partial<PlanFilesDraft>) => void;
};

/** Steps no status maps to and no other executor row uses (plus the row's own). */
function freeSteps(steps: PlanFilesStep[], draft: PlanFilesDraft, own: string): PlanFilesStep[] {
  const mapped = new Set(Object.values(draft.statusSteps));
  const taken = new Set(draft.executorRows.map((row) => row.stepId));
  return steps.filter((step) => step.id === own || (!mapped.has(step.id) && !taken.has(step.id)));
}

type ExecutorRowsProps = {
  draft: PlanFilesDraft;
  steps: PlanFilesStep[];
  disabled: boolean;
  onChange: (rows: ExecutorRow[]) => void;
};

function PlanFilesExecutorRows({ draft, steps, disabled, onChange }: ExecutorRowsProps) {
  const { t } = useTranslation();
  const rows = draft.executorRows;
  const update = (index: number, changes: Partial<ExecutorRow>) =>
    onChange(rows.map((row, i) => (i === index ? { ...row, ...changes } : row)));
  const canAdd = freeSteps(steps, draft, "").length > 0;

  return (
    <div className="space-y-2" data-testid="plan-files-executors">
      <SettingsFieldLabel>{t("planFiles:executorColumns")}</SettingsFieldLabel>
      {rows.map((row, index) => (
        <div key={index} className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <NativeSelect
            id={`plan-files-executor-step-${index}`}
            testId={`plan-files-executor-step-${index}`}
            value={row.stepId}
            disabled={disabled}
            onChange={(stepId) => update(index, { stepId })}
            placeholder={t("planFiles:stepPlaceholder")}
            label={t("planFiles:executorStep")}
          >
            {freeSteps(steps, draft, row.stepId).map((step) => (
              <option key={step.id} value={step.id}>
                {step.name}
              </option>
            ))}
          </NativeSelect>
          <Input
            value={row.name}
            aria-label={t("planFiles:executorName")}
            placeholder={t("planFiles:executorName")}
            data-testid={`plan-files-executor-name-${index}`}
            onChange={(event) => update(index, { name: event.target.value })}
            className={settingsControlClassName("w-full sm:max-w-xs")}
          />
          <button
            type="button"
            aria-label={t("planFiles:removeExecutor")}
            data-testid={`plan-files-executor-remove-${index}`}
            onClick={() => onChange(rows.filter((_, i) => i !== index))}
            className="flex size-6 shrink-0 cursor-pointer items-center justify-center self-end rounded text-muted-foreground hover:bg-muted hover:text-foreground max-md:size-11 sm:self-auto [@media(pointer:coarse)]:size-11"
          >
            <IconX className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || !canAdd}
        onClick={() => onChange([...rows, { stepId: "", name: "" }])}
        className={settingsActionClassName("w-full cursor-pointer sm:w-auto")}
        data-testid="plan-files-add-executor"
      >
        <IconPlus className="mr-2 h-4 w-4" />
        {t("planFiles:addExecutor")}
      </Button>
    </div>
  );
}

/**
 * Executor columns, notes heading, date wake-up, stale threshold and index
 * file settings. Controls stack full width on phones.
 */
export function PlanFilesOperations({ draft, steps, disabled, onChange }: OperationsProps) {
  const { t } = useTranslation();
  return (
    <div className="space-y-4" data-testid="plan-files-operations">
      {draft.workflowId !== "" && (
        <PlanFilesExecutorRows
          draft={draft}
          steps={steps}
          disabled={disabled}
          onChange={(executorRows) => onChange({ executorRows })}
        />
      )}
      <div className={FIELD_ROW}>
        <SettingsFieldLabel htmlFor="plan-files-notes-heading">
          {t("planFiles:notesHeading")}
        </SettingsFieldLabel>
        <Input
          id="plan-files-notes-heading"
          value={draft.notesHeading}
          placeholder={t("planFiles:notesHeadingPlaceholder")}
          onChange={(event) => onChange({ notesHeading: event.target.value })}
          className={settingsControlClassName("w-full sm:max-w-xs")}
        />
      </div>
      <div className="flex min-h-6 items-center gap-2 max-md:min-h-11">
        <Checkbox
          id="plan-files-wake-on-date"
          checked={draft.wakeOnDate}
          onCheckedChange={(value) => onChange({ wakeOnDate: value === true })}
          data-testid="plan-files-wake-on-date"
        />
        <SettingsFieldLabel htmlFor="plan-files-wake-on-date" className="cursor-pointer">
          {t("planFiles:wakeOnDate")}
        </SettingsFieldLabel>
      </div>
      <div className={FIELD_ROW}>
        <SettingsFieldLabel htmlFor="plan-files-stale-days">
          {t("planFiles:staleAfterDays")}
        </SettingsFieldLabel>
        <Input
          id="plan-files-stale-days"
          type="number"
          inputMode="numeric"
          min={0}
          max={365}
          value={draft.staleAfterDays}
          onChange={(event) => onChange({ staleAfterDays: event.target.value })}
          className={settingsControlClassName("w-full sm:max-w-[8rem]")}
        />
      </div>
      <div className={FIELD_ROW}>
        <SettingsFieldLabel htmlFor="plan-files-index-file">
          {t("planFiles:indexFile")}
        </SettingsFieldLabel>
        <Input
          id="plan-files-index-file"
          value={draft.indexFile}
          placeholder={t("planFiles:indexFilePlaceholder")}
          onChange={(event) => onChange({ indexFile: event.target.value })}
          className={settingsControlClassName("w-full sm:max-w-xs")}
        />
      </div>
    </div>
  );
}
