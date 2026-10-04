import { test, expect } from "../../fixtures/test-base";
import { requireBox } from "../../helpers/layout-assertions";
import { MobileKanbanPage } from "../../pages/mobile-kanban-page";
import { localIsoDay, maybeShoot } from "./card-display-hints-helpers";

const DESCRIPTION = "Distinctive phone description text that must stay off the card";

test.describe("Mobile kanban card display hints", () => {
  test.use({ viewport: { width: 375, height: 812 } });

  test("keeps the hint row inside the card at 375px with no horizontal page scroll", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    const mobile = new MobileKanbanPage(testPage);
    await testPage.goto("/");
    const today = await localIsoDay(testPage, 0);
    const task = await apiClient.createTask(
      seedData.workspaceId,
      "Phone hinted card with a title that wraps to two lines",
      {
        workflow_id: seedData.workflowId,
        workflow_step_id: seedData.startStepId,
        description: DESCRIPTION,
        metadata: {
          card_display: {
            date: today,
            date_kind: "deferred",
            executor: { name: "Claude", kind: "agent" },
            progress: { done: 2, total: 5 },
          },
        },
      },
    );
    await mobile.goto();

    const card = mobile.taskCard(task.id);
    await expect(card).toBeVisible();
    const row = card.getByTestId("kanban-card-hint-row");
    await expect(row).toBeVisible();
    await expect(card.getByTestId("kanban-card-date-tag")).toBeVisible();
    await expect(card.getByTestId("kanban-card-progress-chip")).toHaveText("2/5");
    await expect(card.getByTestId("kanban-card-executor-badge")).toBeVisible();
    await expect(card).not.toContainText(DESCRIPTION);

    const cardBox = await requireBox(card, "card");
    const rowBox = await requireBox(row, "hint row");
    expect(rowBox.x).toBeGreaterThanOrEqual(cardBox.x);
    expect(rowBox.y).toBeGreaterThanOrEqual(cardBox.y);
    expect(rowBox.x + rowBox.width).toBeLessThanOrEqual(cardBox.x + cardBox.width);
    expect(rowBox.y + rowBox.height).toBeLessThanOrEqual(cardBox.y + cardBox.height);

    const overflow = await testPage.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
    }));
    expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.innerWidth);

    await maybeShoot(card, "mobile-card.png");
  });
});
