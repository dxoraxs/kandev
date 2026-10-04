"use client";

import { useTask } from "@/hooks/use-task";
import type { Task } from "@/lib/types/http";
import { PlanDecisionBar } from "./plan-decision-bar";

/** Desktop mount: the slim bar under the task top bar. */
export function PlanDecisionDesktopTaskBar({
  task,
}: {
  task: Pick<Task, "id" | "workspace_id" | "workflow_id" | "workflow_step_id">;
}) {
  return (
    <PlanDecisionBar
      taskId={task.id}
      workspaceId={task.workspace_id}
      workflowId={task.workflow_id}
      workflowStepId={task.workflow_step_id}
      presentation="desktop"
    />
  );
}

/** Phone mount: reads the task's board position from the store. */
export function PlanDecisionPhoneBar({ taskId }: { taskId: string }) {
  const task = useTask(taskId);
  if (!task) return null;
  return (
    <PlanDecisionBar
      taskId={taskId}
      workspaceId={task.workspaceId}
      workflowId={task.workflowId}
      workflowStepId={task.workflowStepId}
      presentation="phone"
    />
  );
}
