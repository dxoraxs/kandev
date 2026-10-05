"use client";

import { useTranslation } from "react-i18next";
import {
  IconAlertTriangle,
  IconCalendarDue,
  IconGitCommit,
  IconHourglass,
  IconListCheck,
  IconPlayerPause,
  IconUser,
} from "@tabler/icons-react";
import { cn } from "@/lib/utils";
import {
  dateTagTone,
  localDay,
  type CardFlag,
  type DateKind,
  type DateTone,
} from "@/lib/kanban/card-display";
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

const FLAG_ICON = {
  stale: IconHourglass,
  open_items: IconAlertTriangle,
  uncommitted: IconGitCommit,
} as const;

const FLAG_TEXT_KEY = {
  stale: "kanban:cardFlagStale",
  open_items: "kanban:cardFlagOpenItems",
  uncommitted: "kanban:cardFlagUncommitted",
} as const;

const FLAG_LABEL_KEY = {
  stale: "kanban:cardFlagStaleLabel",
  open_items: "kanban:cardFlagOpenItemsLabel",
  uncommitted: "kanban:cardFlagUncommittedLabel",
} as const;

function FlagTag({ flag }: { flag: CardFlag }) {
  const { t } = useTranslation();
  const label = t(FLAG_LABEL_KEY[flag]);
  const Icon = FLAG_ICON[flag];
  return (
    <span
      className={cn(PILL, TONE_CLASS.warning)}
      data-testid={`kanban-card-flag-${flag}`}
      role="img"
      aria-label={label}
      title={label}
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      <span aria-hidden="true">{t(FLAG_TEXT_KEY[flag])}</span>
    </span>
  );
}

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
    <div
      className="mt-1 flex min-w-0 flex-wrap items-center gap-1"
      data-testid="kanban-card-hint-row"
    >
      {hints.date && <DateTag iso={hints.date.iso} kind={hints.date.kind} />}
      {hints.progress && <ProgressChip done={hints.progress.done} total={hints.progress.total} />}
      {hints.flags?.map((flag) => (
        <FlagTag key={flag} flag={flag} />
      ))}
      {hints.executor && <ExecutorBadge name={hints.executor.name} kind={hints.executor.kind} />}
    </div>
  );
}
