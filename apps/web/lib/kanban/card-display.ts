export type DateKind = "waiting" | "due" | "deferred";
export type DateTone = "neutral" | "warning" | "danger";

export type CardFlag = "stale" | "open_items" | "uncommitted";

/** The flags a card can show, in display order. */
export const CARD_FLAGS: readonly CardFlag[] = ["stale", "open_items", "uncommitted"];

export type CardDisplayHints = {
  date?: { iso: string; kind: DateKind };
  executor?: { name: string; kind: "agent" | "person" };
  progress?: { done: number; total: number };
  flags?: CardFlag[];
};

const ISO_DATE = /^(\d{4})-(\d{2})-(\d{2})$/;
const MAX_EXECUTOR_NAME = 40;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function localDay(iso: string): Date | undefined {
  const match = ISO_DATE.exec(iso);
  if (!match) return undefined;
  const [y, m, d] = [Number(match[1]), Number(match[2]), Number(match[3])];
  const date = new Date(y, m - 1, d);
  const real = date.getFullYear() === y && date.getMonth() === m - 1 && date.getDate() === d;
  return real ? date : undefined;
}

function parseDate(raw: Record<string, unknown>): CardDisplayHints["date"] {
  if (typeof raw.date !== "string" || !localDay(raw.date)) return undefined;
  const kind = raw.date_kind === "waiting" || raw.date_kind === "deferred" ? raw.date_kind : "due";
  return { iso: raw.date, kind };
}

function parseExecutor(raw: unknown): CardDisplayHints["executor"] {
  if (!isRecord(raw) || typeof raw.name !== "string") return undefined;
  const name = raw.name.trim();
  const chars = Array.from(name).length;
  if (chars < 1 || chars > MAX_EXECUTOR_NAME) return undefined;
  return { name, kind: raw.kind === "person" ? "person" : "agent" };
}

function parseProgress(raw: unknown): CardDisplayHints["progress"] {
  if (!isRecord(raw)) return undefined;
  const { done, total } = raw;
  if (!Number.isInteger(done) || !Number.isInteger(total)) return undefined;
  const d = done as number;
  const t = total as number;
  return t > 0 && d >= 0 && d <= t ? { done: d, total: t } : undefined;
}

function parseFlags(raw: unknown): CardDisplayHints["flags"] {
  if (!Array.isArray(raw)) return undefined;
  const flags = CARD_FLAGS.filter((flag) => raw.includes(flag));
  return flags.length > 0 ? flags : undefined;
}

/** Validates each field independently; returns undefined when none is valid. */
export function cardDisplayFromMetadata(metadata: unknown): CardDisplayHints | undefined {
  if (!isRecord(metadata) || !isRecord(metadata.card_display)) return undefined;
  const raw = metadata.card_display;
  const hints: CardDisplayHints = {};
  const date = parseDate(raw);
  const executor = parseExecutor(raw.executor);
  const progress = parseProgress(raw.progress);
  const flags = parseFlags(raw.flags);
  if (date) hints.date = date;
  if (executor) hints.executor = executor;
  if (progress) hints.progress = progress;
  if (flags) hints.flags = flags;
  return date || executor || progress || flags ? hints : undefined;
}

const DAY_MS = 24 * 60 * 60 * 1000;

/** Compares local calendar days; `deferred` never reads as overdue. */
export function dateTagTone(iso: string, kind: DateKind, today: Date): DateTone {
  const target = localDay(iso);
  if (!target) return "neutral";
  const base = new Date(today.getFullYear(), today.getMonth(), today.getDate());
  const diffDays = Math.round((target.getTime() - base.getTime()) / DAY_MS);
  if (diffDays < 0) return kind === "deferred" ? "warning" : "danger";
  return diffDays <= 1 ? "warning" : "neutral";
}
