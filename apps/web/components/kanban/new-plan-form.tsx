"use client";

import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { controlSizingClassName } from "@kandev/ui/control-sizing";
import { Input } from "@kandev/ui/input";
import { Textarea } from "@kandev/ui/textarea";
import { cn } from "@/lib/utils";
import type { CreatePlanState } from "@/hooks/domains/plans/use-create-plan";
import type { CreatePlanBody } from "@/lib/api/domains/plan-files-api";
import { TASK_PRIORITY_LABEL_KEYS, TASK_PRIORITY_TOKENS } from "@/lib/tasks/task-priority";
import type { Repository, TaskPriority } from "@/lib/types/http";

/** Mirrors the server rule for a plan file name. */
const FILE_NAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]*\.md$/;
const EXECUTOR_NONE = "none";
const EXECUTOR_PREFIX = "name:";
const SELECT_CLASS = "w-full rounded-md border bg-background px-2 text-xs";

export type NewPlanFormProps = {
  repositories: Repository[];
  directories: string[];
  /** Distinct executor names of the plan board; the field is hidden when empty. */
  executors: string[];
  create: CreatePlanState;
  onClose: () => void;
  /** Phone composition: scroll region with a sticky footer and full-width touch controls. */
  touch: boolean;
};

function Field({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="text-xs font-medium">
        {label}
      </label>
      {children}
    </div>
  );
}

function OptionSelect({
  id,
  value,
  onChange,
  children,
}: {
  id: string;
  value: string;
  onChange: (value: string) => void;
  children: ReactNode;
}) {
  return (
    <select
      id={id}
      data-testid={id}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className={cn(controlSizingClassName("standard"), SELECT_CLASS)}
    >
      {children}
    </select>
  );
}

type FieldValues = {
  repositoryId: string;
  directory: string;
  title: string;
  fileName: string;
  priority: TaskPriority;
  executor: string;
  body: string;
};

function buildBody(fields: FieldValues): CreatePlanBody {
  const body: CreatePlanBody = {
    repository_id: fields.repositoryId,
    directory: fields.directory,
    title: fields.title.trim(),
    priority: fields.priority,
  };
  const fileName = fields.fileName.trim();
  if (fileName) body.file_name = fileName;
  if (fields.executor.startsWith(EXECUTOR_PREFIX)) {
    body.executor = fields.executor.slice(EXECUTOR_PREFIX.length);
  }
  if (fields.body.trim()) body.body = fields.body;
  return body;
}

/** Field state of the form; empty repository and directory choices read as the first option. */
function useNewPlanFields(repositories: Repository[], directories: string[]) {
  const [repositoryChoice, setRepository] = useState("");
  const [directoryChoice, setDirectory] = useState("");
  const [title, setTitle] = useState("");
  const [fileName, setFileName] = useState("");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [executor, setExecutor] = useState(EXECUTOR_NONE);
  const [body, setBody] = useState("");
  const values: FieldValues = {
    repositoryId: repositoryChoice || (repositories[0]?.id ?? ""),
    directory: directoryChoice || (directories[0] ?? ""),
    title,
    fileName,
    priority,
    executor,
    body,
  };
  const fileNameInvalid = fileName.trim() !== "" && !FILE_NAME_PATTERN.test(fileName.trim());
  const ready =
    title.trim() !== "" &&
    values.repositoryId !== "" &&
    values.directory !== "" &&
    !fileNameInvalid;
  return {
    values,
    fileNameInvalid,
    ready,
    set: { setRepository, setDirectory, setTitle, setFileName, setPriority, setExecutor, setBody },
  };
}

type Fields = ReturnType<typeof useNewPlanFields>;

function LocationFields({
  fields,
  repositories,
  directories,
}: {
  fields: Fields;
  repositories: Repository[];
  directories: string[];
}) {
  const { t } = useTranslation();
  return (
    <>
      {repositories.length === 0 ? (
        <p className="text-xs text-muted-foreground" data-testid="new-plan-no-repository">
          {t("planFiles:newPlanNoRepository")}
        </p>
      ) : null}
      <Field id="new-plan-repository" label={t("planFiles:newPlanRepository")}>
        <OptionSelect
          id="new-plan-repository"
          value={fields.values.repositoryId}
          onChange={fields.set.setRepository}
        >
          {repositories.map((repo) => (
            <option key={repo.id} value={repo.id}>
              {repo.name}
            </option>
          ))}
        </OptionSelect>
      </Field>
      <Field id="new-plan-directory" label={t("planFiles:newPlanDirectory")}>
        <OptionSelect
          id="new-plan-directory"
          value={fields.values.directory}
          onChange={fields.set.setDirectory}
        >
          {directories.map((dir) => (
            <option key={dir} value={dir}>
              {dir}
            </option>
          ))}
        </OptionSelect>
      </Field>
    </>
  );
}

function NameFields({ fields, full }: { fields: Fields; full: string | undefined }) {
  const { t } = useTranslation();
  return (
    <>
      <Field id="new-plan-title" label={t("planFiles:newPlanName")}>
        <Input
          id="new-plan-title"
          data-testid="new-plan-title"
          value={fields.values.title}
          onChange={(event) => fields.set.setTitle(event.target.value)}
          className={full}
        />
      </Field>
      <Field id="new-plan-file-name" label={t("planFiles:newPlanFileName")}>
        <Input
          id="new-plan-file-name"
          data-testid="new-plan-file-name"
          value={fields.values.fileName}
          onChange={(event) => fields.set.setFileName(event.target.value)}
          placeholder={t("planFiles:newPlanFileNamePlaceholder")}
          aria-invalid={fields.fileNameInvalid}
          className={full}
        />
        {fields.fileNameInvalid ? (
          <p className="text-xs text-destructive" data-testid="new-plan-file-name-error">
            {t("planFiles:newPlanFileNameInvalid")}
          </p>
        ) : null}
      </Field>
    </>
  );
}

function OptionFields({
  fields,
  executors,
  touch,
}: {
  fields: Fields;
  executors: string[];
  touch: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div className={cn("grid gap-3", touch ? "grid-cols-1" : "grid-cols-2")}>
      <Field id="new-plan-priority" label={t("planFiles:newPlanPriority")}>
        <OptionSelect
          id="new-plan-priority"
          value={fields.values.priority}
          onChange={(value) => fields.set.setPriority(value as TaskPriority)}
        >
          {TASK_PRIORITY_TOKENS.map((token) => (
            <option key={token} value={token}>
              {t(TASK_PRIORITY_LABEL_KEYS[token])}
            </option>
          ))}
        </OptionSelect>
      </Field>
      {executors.length > 0 ? (
        <Field id="new-plan-executor" label={t("planFiles:newPlanExecutor")}>
          <OptionSelect
            id="new-plan-executor"
            value={fields.values.executor}
            onChange={fields.set.setExecutor}
          >
            <option value={EXECUTOR_NONE}>{t("planFiles:newPlanExecutorNone")}</option>
            {executors.map((name) => (
              <option key={name} value={EXECUTOR_PREFIX + name}>
                {name}
              </option>
            ))}
          </OptionSelect>
        </Field>
      ) : null}
    </div>
  );
}

function FormFields(props: NewPlanFormProps & { fields: Fields }) {
  const { t } = useTranslation();
  const { fields, repositories, directories, executors, create, touch } = props;
  return (
    <div className="space-y-3">
      <LocationFields fields={fields} repositories={repositories} directories={directories} />
      <NameFields fields={fields} full={touch ? "w-full" : undefined} />
      <OptionFields fields={fields} executors={executors} touch={touch} />
      <Field id="new-plan-body" label={t("planFiles:newPlanBody")}>
        <Textarea
          id="new-plan-body"
          data-testid="new-plan-body"
          value={fields.values.body}
          onChange={(event) => fields.set.setBody(event.target.value)}
          rows={touch ? 8 : 5}
        />
      </Field>
      {create.error ? (
        <p role="alert" className="text-xs text-destructive" data-testid="new-plan-error">
          {create.error}
        </p>
      ) : null}
    </div>
  );
}

function FormActions({
  touch,
  disabled,
  onCancel,
  onConfirm,
}: {
  touch: boolean;
  disabled: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const { t } = useTranslation();
  const buttonClass = cn("cursor-pointer", touch && "min-h-11 w-full");
  return (
    <>
      <Button
        variant="outline"
        className={buttonClass}
        onClick={onCancel}
        data-testid="new-plan-cancel"
      >
        {t("planFiles:newPlanCancel")}
      </Button>
      <Button
        className={buttonClass}
        disabled={disabled}
        onClick={onConfirm}
        data-testid="new-plan-create"
      >
        {t("planFiles:newPlanCreate")}
      </Button>
    </>
  );
}

/** The New plan fields and actions, shared by the desktop dialog and the phone surface. */
export function NewPlanForm(props: NewPlanFormProps) {
  const { create, onClose, touch } = props;
  const fields = useNewPlanFields(props.repositories, props.directories);
  const confirm = async () => {
    if (await create.submit(buildBody(fields.values))) onClose();
  };
  const actions = (
    <FormActions
      touch={touch}
      disabled={create.pending || !fields.ready}
      onCancel={onClose}
      onConfirm={confirm}
    />
  );
  if (!touch) {
    return (
      <>
        <FormFields {...props} fields={fields} />
        <div className="flex justify-end gap-2">{actions}</div>
      </>
    );
  }
  return (
    <>
      <div
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-4"
        data-testid="new-plan-scroll"
      >
        <FormFields {...props} fields={fields} />
      </div>
      <div
        className="sticky bottom-0 flex flex-col gap-2 border-t bg-background px-4 pt-3"
        style={{ paddingBottom: "calc(0.75rem + env(safe-area-inset-bottom, 0px))" }}
        data-testid="new-plan-footer"
      >
        {actions}
      </div>
    </>
  );
}
