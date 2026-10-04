"use client";

import { useTranslation } from "react-i18next";
import {
  IconCalendarDue,
  IconHourglass,
  IconListCheck,
  IconPlayerPause,
  IconUser,
} from "@tabler/icons-react";
import { cn } from "@/lib/utils";
import { dateTagTone, localDay, type DateKind, type DateTone } from "@/lib/kanban/card-display";
import type { Task } from "@/components/kanban-card";

const PILL =
  "inline-flex h-5 shrink-0 items-center gap-1 rounded-full px-1.5 text-[11px] font-medium";

const TONE_CLASS: Record<DateTone, string> = {
  neutral: "bg-muted text-foreground",
  warning: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  danger: "bg-red-500/15 text-red-600 dark:text-red-400",
};

const DATE_ICON = {
  waiting: IconHourglass,
  due: IconCalendarDue,
  deferred: IconPlayerPause,
} as const;

const DATE_KEY = {
  waiting: "kanban:cardDateWaiting",
  due: "kanban:cardDateDue",
  deferred: "kanban:cardDateDeferred",
} as const;

function DateTag({ iso, kind }: { iso: string; kind: DateKind }) {
  const { t, i18n } = useTranslation();
  const today = new Date();
  const day = localDay(iso);
  if (!day) return null;
  const short = new Intl.DateTimeFormat(i18n.language, {
    day: "numeric",
    month: "short",
    ...(day.getFullYear() === today.getFullYear() ? {} : { year: "numeric" }),
  }).format(day);
  const full = t(DATE_KEY[kind], {
    date: new Intl.DateTimeFormat(i18n.language, { dateStyle: "long" }).format(day),
  });
  const tone = dateTagTone(iso, kind, today);
  const Icon = DATE_ICON[kind];
  return (
    <span
      className={cn(PILL, TONE_CLASS[tone])}
      data-testid="kanban-card-date-tag"
      data-tone={tone}
      role="img"
      aria-label={full}
      title={full}
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      <span aria-hidden="true">{short}</span>
    </span>
  );
}

function ProgressChip({ done, total }: { done: number; total: number }) {
  const { t } = useTranslation();
  const complete = done === total;
  const label = t("kanban:cardProgress", { count: total, done });
  return (
    <span
      className={cn(
        PILL,
        complete ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300" : TONE_CLASS.neutral,
      )}
      data-testid="kanban-card-progress-chip"
      data-complete={complete ? "true" : "false"}
      role="img"
      aria-label={label}
      title={label}
    >
      <IconListCheck className="h-3 w-3" aria-hidden="true" />
      <span aria-hidden="true">{`${done}/${total}`}</span>
    </span>
  );
}

function ExecutorBadge({ name, kind }: { name: string; kind: "agent" | "person" }) {
  const { t } = useTranslation();
  const label = t("kanban:cardExecutor", { name });
  const isPerson = kind === "person";
  return (
    <span
      className={cn(
        "ml-auto inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full text-[10px] font-semibold",
        isPerson ? "bg-muted text-foreground" : "bg-primary/15 text-primary",
      )}
      data-testid="kanban-card-executor-badge"
      data-kind={kind}
      role="img"
      aria-label={label}
      title={label}
    >
      {isPerson ? (
        <IconUser className="h-3 w-3" aria-hidden="true" />
      ) : (
        <span aria-hidden="true">{Array.from(name)[0]?.toUpperCase()}</span>
      )}
    </span>
  );
}

export function KanbanCardHintRow({ task }: { task: Task }) {
  const hints = task.cardDisplay;
  if (!hints) return null;
  return (
    <div className="mt-1 flex min-w-0 items-center gap-1" data-testid="kanban-card-hint-row">
      {hints.date && <DateTag iso={hints.date.iso} kind={hints.date.kind} />}
      {hints.progress && <ProgressChip done={hints.progress.done} total={hints.progress.total} />}
      {hints.executor && <ExecutorBadge name={hints.executor.name} kind={hints.executor.kind} />}
    </div>
  );
}
