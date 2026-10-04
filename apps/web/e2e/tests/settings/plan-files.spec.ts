import { test, expect } from "../../fixtures/test-base";
import { KanbanPage } from "../../pages/kanban-page";
import {
  PLAN_FILES_ENV,
  createPlanRepository,
  dragCardToColumn,
  enablePlanFilesAndCreateBoard,
} from "../../helpers/plan-files";

test.describe("Plan files", () => {
  test.describe.configure({ mode: "serial" });

  test("syncs a plan file to the Plans board, follows edits and writes moves back", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    try {
      const plan = await createPlanRepository({
        backend,
        apiClient,
        workspaceId: seedData.workspaceId,
        title: "Plan files e2e original",
      });

      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/workflows`);
      const save = await enablePlanFilesAndCreateBoard(testPage);
      await expect(save).toBeEnabled();
      await save.click();
      const syncNow = testPage.getByTestId("plan-files-sync-now");
      await expect(syncNow).toBeEnabled();
      await syncNow.click();
      await expect(testPage.getByTestId("plan-files-status")).toHaveAttribute("data-state", "ok");

      // AC-TASKS-PLAN-FILES-002.2: the file becomes a task in Queue.
      const boardId = await testPage.getByTestId("plan-files-board").inputValue();
      const steps = (await apiClient.listWorkflowSteps(boardId)).steps;
      const stepId = (name: string) => {
        const step = steps.find((candidate) => candidate.name === name);
        if (!step) throw new Error(`Plans board has no ${name} step`);
        return step.id;
      };
      // Wide enough that all seven columns are on screen, so a drag never scrolls the board.
      await testPage.setViewportSize({ width: 2400, height: 900 });
      const kanban = new KanbanPage(testPage);
      await kanban.goto(boardId);
      await expect(
        kanban.taskCardInColumn("Plan files e2e original", stepId("Queue")),
      ).toBeVisible();

      // AC-TASKS-PLAN-FILES-003.1: an edit on disk updates the task title.
      plan.writePlan("Plan files e2e renamed", "queued");
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/workflows`);
      await testPage.getByTestId("plan-files-sync-now").click();
      await expect(testPage.getByTestId("plan-files-counts")).toContainText("1 updated");
      await kanban.goto(boardId);
      await expect(
        kanban.taskCardInColumn("Plan files e2e renamed", stepId("Queue")),
      ).toBeVisible();

      // AC-TASKS-PLAN-FILES-005.1: a board move is written back to the file.
      const card = kanban.taskCardInColumn("Plan files e2e renamed", stepId("Queue"));
      await dragCardToColumn(testPage, card, kanban.columnByStepId(stepId("Done")));
      await expect(
        kanban.taskCardInColumn("Plan files e2e renamed", stepId("Done")),
      ).toBeVisible();
      await expect.poll(() => plan.readPlan()).toContain("board: done");
    } finally {
      await release();
    }
  });
});
