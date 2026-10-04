"use client";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@kandev/ui/select";
import { useTranslation } from "react-i18next";
import { useKanbanDisplaySettings } from "@/hooks/use-kanban-display-settings";
import {
  KANBAN_SORT_LABEL_KEYS,
  KANBAN_SORT_OPTIONS,
  type KanbanSort,
} from "@/lib/kanban/kanban-sort";

/**
 * Board sort picker for the top bar. It reads and writes the same persisted
 * `kanbanSort` setting as the "Sort" group inside the display dropdown, through
 * the same `useKanbanDisplaySettings` handler, so the two never disagree.
 */
export function KanbanHeaderSort() {
  const { t } = useTranslation();
  const { boardSort, onBoardSortChange } = useKanbanDisplaySettings();
  return (
    <Select value={boardSort} onValueChange={(value) => onBoardSortChange(value as KanbanSort)}>
      <SelectTrigger
        size="sm"
        data-testid="header-board-sort"
        aria-label={t("kanban:boardSort")}
        className="h-8 w-44 cursor-pointer"
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {KANBAN_SORT_OPTIONS.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {t(KANBAN_SORT_LABEL_KEYS[option.value])}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
