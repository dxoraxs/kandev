"use client";

import { useAppStore } from "@/components/state-provider";
import { useTask } from "@/hooks/use-task";
import {
  SessionlessTaskView,
  TaskDescriptionDocument,
  useSessionlessTaskView,
} from "./sessionless-task-view";

/**
 * Body of the Description tab: the active task's description as a document.
 * While the task has no session it also carries the corner status.
 */
export function TaskDescriptionPanel() {
  const taskId = useAppStore((state) => state.tasks.activeTaskId);
  const hasSession = useAppStore((state) =>
    taskId ? (state.taskSessionsByTask.itemsByTaskId[taskId]?.length ?? 0) > 0 : false,
  );
  const sessionlessView = useSessionlessTaskView({ taskId, hasSession, launchErrorOwned: false });
  const task = useTask(taskId);
  if (!taskId) return null;
  if (sessionlessView) return <SessionlessTaskView {...sessionlessView} />;
  const description = task?.description?.trim() ? task.description : null;
  return <TaskDescriptionDocument taskId={taskId} description={description} />;
}
