import { cardDisplayFromMetadata } from "@/lib/kanban/card-display";
import type { HydrationState } from "../store";

type KanbanSnapshots = NonNullable<HydrationState["kanbanMulti"]>["snapshots"] | undefined;

/** Boot-payload tasks carry raw metadata; derive the card hints the same way the WS mapper does. */
export function withCardDisplay<T extends { metadata?: unknown }>(
  tasks: T[] | undefined,
): T[] | undefined {
  return tasks?.map((task) => ({ ...task, cardDisplay: cardDisplayFromMetadata(task.metadata) }));
}

export function withSnapshotCardDisplay(snapshots: KanbanSnapshots): KanbanSnapshots {
  if (!snapshots) return snapshots;
  return Object.fromEntries(
    Object.entries(snapshots).map(([id, snapshot]) => [
      id,
      snapshot ? { ...snapshot, tasks: withCardDisplay(snapshot.tasks) } : snapshot,
    ]),
  ) as KanbanSnapshots;
}
