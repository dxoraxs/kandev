"use client";

import { useTranslation } from "react-i18next";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@kandev/ui/table";
import Link from "@/components/routing/app-link";
import { useRouter } from "@/lib/routing/client-router";
import { linkToTask } from "@/lib/links";
import type { WaitingOwnerItem } from "@/lib/api/domains/plan-files-api";
import { formatWaitingDate } from "./waiting-owner-date";

/** Desktop composition: one table row per plan; the whole row opens the task. */
export function WaitingOwnerTable({ items }: { items: WaitingOwnerItem[] }) {
  const { t, i18n } = useTranslation();
  const router = useRouter();
  return (
    <div
      className="overflow-hidden rounded-lg border border-border"
      data-testid="plans-waiting-table-wrap"
    >
      <Table data-testid="plans-waiting-table" className="text-sm">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>{t("planFiles:waitingColumnWorkspace")}</TableHead>
            <TableHead>{t("planFiles:waitingColumnPlan")}</TableHead>
            <TableHead>{t("planFiles:waitingColumnRepository")}</TableHead>
            <TableHead>{t("planFiles:waitingColumnDate")}</TableHead>
            <TableHead>{t("planFiles:waitingColumnExecutor")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => {
            const date = formatWaitingDate(item.date, i18n.language);
            return (
              <TableRow
                key={item.task_id}
                data-testid={`plans-waiting-row-${item.task_id}`}
                className="cursor-pointer"
                onClick={() => router.push(linkToTask(item.task_id))}
              >
                <TableCell>{item.workspace_name}</TableCell>
                <TableCell className="font-medium">
                  <Link
                    href={linkToTask(item.task_id)}
                    className="cursor-pointer hover:underline"
                    onClick={(event) => event.stopPropagation()}
                  >
                    {item.title}
                  </Link>
                </TableCell>
                <TableCell>{item.repository_name}</TableCell>
                <TableCell>
                  {date && <span data-testid="plans-waiting-date">{date}</span>}
                </TableCell>
                <TableCell>{item.executor}</TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
