import { localDay } from "@/lib/kanban/card-display";

/** Short local date such as "5 Oct" for a YYYY-MM-DD value; empty when it is not a real date. */
export function formatWaitingDate(iso: string, language: string, today = new Date()): string {
  const day = localDay(iso);
  if (!day) return "";
  return new Intl.DateTimeFormat(language, {
    day: "numeric",
    month: "short",
    ...(day.getFullYear() === today.getFullYear() ? {} : { year: "numeric" }),
  }).format(day);
}
