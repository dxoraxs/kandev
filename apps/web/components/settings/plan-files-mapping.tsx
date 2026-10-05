"use client";

import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { IconLoader2, IconPlus } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { settingsActionClassName, settingsControlClassName } from "./settings-control";
import { SettingsFieldLabel } from "./settings-typography";
import {
  PLAN_FILE_STATUSES,
  type PlanFileStatus,
  type PlanFileStatusSteps,
} from "@/lib/api/domains/plan-files-api";
import type { PlanFilesStep, PlanFilesWorkflow } from "@/hooks/domains/settings/use-plan-files";

// Protocol tokens are sent verbatim; only the label is translated. The values
// are catalog keys resolved at render.
const STATUS_LABEL_KEYS: Record<PlanFileStatus, string> = {
  queued: "planFiles:statusQueued",
  in_progress: "planFiles:statusInProgress",
  waiting_owner: "planFiles:statusWaitingOwner",
  waiting_external: "planFiles:statusWaitingExternal",
  deferred: "planFiles:statusDeferred",
  done: "planFiles:statusDone",
};

const SELECT_CLASS = "w-full rounded-md border bg-background px-2";

type NativeSelectProps = {
  id: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  children: ReactNode;
  disabled?: boolean;
  testId?: string;
  /** Accessible name for a select that has no visible label. */
  label?: string;
};

export function NativeSelect({
  id,
  value,
  onChange,
  placeholder,
  children,
  disabled,
  testId,
  label,
}: NativeSelectProps) {
  return (
    <select
      id={id}
      value={value}
      disabled={disabled}
      aria-label={label}
      data-testid={testId}
      onChange={(event) => onChange(event.target.value)}
      className={settingsControlClassName(SELECT_CLASS)}
    >
      <option value="">{placeholder}</option>
      {children}
    </select>
  );
}

type BoardFieldProps = {
  workflows: PlanFilesWorkflow[];
  workflowId: string;
  creating: boolean;
  onSelect: (workflowId: string) => void;
  onCreate: () => void;
};

/** Board select with the Create Plans board action (stacked on phones). */
export function PlanFilesBoardField({
  workflows,
  workflowId,
  creating,
  onSelect,
  onCreate,
}: BoardFieldProps) {
  const { t } = useTranslation();
  return (
    <div className="grid gap-1.5 sm:grid-cols-[8rem_minmax(0,1fr)_auto] sm:items-center sm:gap-3">
      <SettingsFieldLabel htmlFor="plan-files-board">{t("planFiles:board")}</SettingsFieldLabel>
      <NativeSelect
        id="plan-files-board"
        testId="plan-files-board"
        value={workflowId}
        onChange={onSelect}
        placeholder={t("planFiles:boardPlaceholder")}
      >
        {workflows.map((workflow) => (
          <option key={workflow.id} value={workflow.id}>
            {workflow.name}
          </option>
        ))}
      </NativeSelect>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={creating}
        onClick={onCreate}
        className={settingsActionClassName("w-full cursor-pointer sm:w-auto")}
        data-testid="plan-files-create-board"
      >
        {creating ? (
          <IconLoader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <IconPlus className="mr-2 h-4 w-4" />
        )}
        {t("planFiles:createBoard")}
      </Button>
    </div>
  );
}

type MappingProps = {
  steps: PlanFilesStep[];
  statusSteps: PlanFileStatusSteps;
  disabled: boolean;
  onChange: (status: PlanFileStatus, stepId: string) => void;
};

/** One label-over-select row per status; `hidden` is never mapped. */
export function PlanFilesMapping({ steps, statusSteps, disabled, onChange }: MappingProps) {
  const { t } = useTranslation();
  return (
    <div className="space-y-3" data-testid="plan-files-mapping">
      {PLAN_FILE_STATUSES.map((status) => {
        const id = `plan-files-step-${status}`;
        return (
          <div
            key={status}
            className="grid gap-1.5 sm:grid-cols-[8rem_minmax(0,1fr)] sm:items-center sm:gap-3"
          >
            <SettingsFieldLabel htmlFor={id}>{t(STATUS_LABEL_KEYS[status])}</SettingsFieldLabel>
            <NativeSelect
              id={id}
              testId={id}
              disabled={disabled}
              value={statusSteps[status] ?? ""}
              onChange={(stepId) => onChange(status, stepId)}
              placeholder={t("planFiles:stepPlaceholder")}
            >
              {steps.map((step) => (
                <option key={step.id} value={step.id}>
                  {step.name}
                </option>
              ))}
            </NativeSelect>
          </div>
        );
      })}
    </div>
  );
}
