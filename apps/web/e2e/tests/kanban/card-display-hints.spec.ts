import { test, expect } from "../../fixtures/test-base";
import { KanbanPage } from "../../pages/kanban-page";
import { backgroundColor, localIsoDay, maybeShoot } from "./card-display-hints-helpers";

const DESCRIPTION = "Distinctive description text that must stay off the card";

test.describe("Kanban card display hints", () => {
  test("renders date tag, progress chip and executor badge without a description", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    const kanban = new KanbanPage(testPage);
    await testPage.goto("/");
    const today = await localIsoDay(testPage, 0);
    const task = await apiClient.createTask(seedData.workspaceId, "Hinted card task", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      description: DESCRIPTION,
      metadata: {
        card_display: {
          date: today,
          date_kind: "waiting",
          executor: { name: "Claude", kind: "agent" },
          progress: { done: 1, total: 3 },
        },
      },
    });
    await kanban.goto();

    const card = kanban.taskCard(task.id);
    await expect(card).toBeVisible();
    await expect(card.getByTestId("kanban-card-hint-row")).toBeVisible();

    const dateTag = card.getByTestId("kanban-card-date-tag");
    await expect(dateTag).toHaveAttribute("data-tone", "warning");
    await expect(dateTag).toHaveAttribute("aria-label", /^Waiting until .+\d{4}$/);
    await expect(dateTag).toHaveAttribute("title", /^Waiting until /);

    const chip = card.getByTestId("kanban-card-progress-chip");
    await expect(chip).toHaveText("1/3");
    await expect(chip).toHaveAttribute("data-complete", "false");
    await expect(chip).toHaveAttribute("aria-label", "1 of 3 steps done");

    const badge = card.getByTestId("kanban-card-executor-badge");
    await expect(badge).toHaveText("C");
    await expect(badge).toHaveAttribute("data-kind", "agent");
    await expect(badge).toHaveAttribute("aria-label", "Executor: Claude");

    await expect(card).not.toContainText(DESCRIPTION);

    await maybeShoot(card, "desktop-card.png");
  });

  test("date tag background differs from the card background in neutral and danger tones", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    const kanban = new KanbanPage(testPage);
    await testPage.goto("/");
    const future = await localIsoDay(testPage, 30);
    const past = await localIsoDay(testPage, -1);
    const neutral = await apiClient.createTask(seedData.workspaceId, "Neutral date task", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      metadata: { card_display: { date: future } },
    });
    const overdue = await apiClient.createTask(seedData.workspaceId, "Overdue date task", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      metadata: { card_display: { date: past, date_kind: "due" } },
    });
    await kanban.goto();

    for (const [task, tone] of [
      [neutral, "neutral"],
      [overdue, "danger"],
    ] as const) {
      const card = kanban.taskCard(task.id);
      const tag = card.getByTestId("kanban-card-date-tag");
      await expect(tag).toHaveAttribute("data-tone", tone);
      const tagBg = await backgroundColor(tag);
      const cardBg = await backgroundColor(card);
      expect(tagBg, `${tone} tag background must not be transparent`).not.toBe("rgba(0, 0, 0, 0)");
      expect(tagBg, `${tone} tag background must differ from the card`).not.toBe(cardBg);
    }
  });

  test("switches the date tag to the danger tone after a metadata update without reload", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    const kanban = new KanbanPage(testPage);
    await testPage.goto("/");
    const future = await localIsoDay(testPage, 30);
    const past = await localIsoDay(testPage, -1);
    const task = await apiClient.createTask(seedData.workspaceId, "Live update task", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      metadata: { card_display: { date: future } },
    });
    await kanban.goto();

    const tag = kanban.taskCard(task.id).getByTestId("kanban-card-date-tag");
    await expect(tag).toHaveAttribute("data-tone", "neutral");

    await apiClient.updateTaskMetadata(task.id, { card_display: { date: past } });
    await expect(tag).toHaveAttribute("data-tone", "danger");
  });

  test("hovering the date tag neither navigates nor starts a drag", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    const kanban = new KanbanPage(testPage);
    await testPage.goto("/");
    const today = await localIsoDay(testPage, 0);
    const task = await apiClient.createTask(seedData.workspaceId, "Hover task", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      metadata: { card_display: { date: today } },
    });
    await kanban.goto();

    const card = kanban.taskCard(task.id);
    const tag = card.getByTestId("kanban-card-date-tag");
    const urlBefore = testPage.url();
    const boxBefore = await card.boundingBox();
    await tag.hover();
    await expect(tag).toBeVisible();
    expect(testPage.url()).toBe(urlBefore);
    const boxAfter = await card.boundingBox();
    expect(boxAfter).toEqual(boxBefore);
    const willChange = await card.evaluate((node) => getComputedStyle(node).willChange);
    expect(willChange).not.toBe("transform");
  });

  test("a task without card_display shows no hint elements", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    const kanban = new KanbanPage(testPage);
    const task = await apiClient.createTask(seedData.workspaceId, "Plain card task", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      description: DESCRIPTION,
    });
    await kanban.goto();

    const card = kanban.taskCard(task.id);
    await expect(card).toBeVisible();
    await expect(card.getByTestId("kanban-card-hint-row")).toHaveCount(0);
    await expect(card.getByTestId("kanban-card-date-tag")).toHaveCount(0);
    await expect(card.getByTestId("kanban-card-progress-chip")).toHaveCount(0);
    await expect(card.getByTestId("kanban-card-executor-badge")).toHaveCount(0);
    await expect(card).not.toContainText(DESCRIPTION);
  });
});
