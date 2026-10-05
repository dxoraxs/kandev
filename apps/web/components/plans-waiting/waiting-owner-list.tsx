"use client";

import { useTranslation } from "react-i18next";
import { IconChevronRight } from "@tabler/icons-react";
import Link from "@/components/routing/app-link";
import { linkToTask } from "@/lib/links";
import type { WaitingOwnerItem } from "@/lib/api/domains/plan-files-api";
import { formatWaitingDate } from "./waiting-owner-date";

/** Phone composition: one full-width row per plan, title then a details line. */
export function WaitingOwnerList({ items }: { items: WaitingOwnerItem[] }) {
  const { i18n } = useTranslation();
  return (
    <ul className="flex flex-col gap-2" data-testid="plans-waiting-list">
      {items.map((item) => {
        const details = [
          item.workspace_name,
          formatWaitingDate(item.date, i18n.language),
          item.executor,
        ]
          .filter(Boolean)
          .join(" · ");
        return (
          <li key={item.task_id}>
            <Link
              href={linkToTask(item.task_id)}
              data-testid={`plans-waiting-row-${item.task_id}`}
              className="flex min-h-11 w-full cursor-pointer items-center gap-3 rounded-lg border border-border px-3 py-2"
            >
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="truncate text-sm font-medium">{item.title}</span>
                <span
                  className="truncate text-xs text-muted-foreground"
                  data-testid="plans-waiting-meta"
                >
                  {details}
                </span>
              </span>
              <IconChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
