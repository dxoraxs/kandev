import type { Locator } from "@playwright/test";
import { test, expect } from "../../fixtures/test-base";
import { waitForHttp } from "../../helpers/causal-waits";
import { settledBoundingBox } from "../../helpers/settled-box";
import { MobileKanbanPage } from "../../pages/mobile-kanban-page";
import {
  PLAN_FILES_ENV,
  configurePlanBoard,
  createPlanFilesRepository,
  localDay,
  planFileText,
  planOpsShot,
  planTaskId,
  type PlanFilesRepository,
} from "../../helpers/plan-files";

const PLANS_DIR = "docs/plans";
const MIN_TOUCH_TARGET = 44;
// A full-width control spans its container minus the side insets of a bottom sheet (two 24px).
const INSETS = 49;

/** The control is at least 44px tall and spans its container's width minus the insets. */
async function expectFullWidthTouchTarget(control: Locator, container: Locator, label: string) {
  const box = await settledBoundingBox(control);
  const containerBox = await settledBoundingBox(container);
  expect(box.height, `${label} height`).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
  expect(box.width, `${label} width`).toBeGreaterThanOrEqual(containerBox.width - INSETS);
}

const seeded: PlanFilesRepository[] = [];

/** Creates a plan repository that the test unregisters when it ends. */
async function seedRepo(options: Parameters<typeof createPlanFilesRepository>[0]) {
  const repo = await createPlanFilesRepository(options);
  seeded.push(repo);
  return repo;
}

test.describe("Mobile plan board operations", () => {
  test.afterEach(async () => {
    for (const repo of seeded.splice(0)) await repo.cleanup();
  });

  test("returns a waiting plan from the decision drawer with full-width touch controls", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    try {
      const today = localDay();
      const repo = await seedRepo({
        backend,
        apiClient,
        workspaceId: seedData.workspaceId,
        name: "plan-ops-phone-decision",
        files: {
          [`${PLANS_DIR}/waiting.md`]: planFileText({
            title: "Phone plan awaiting owner",
            board: "waiting_owner",
            date: today,
          }),
        },
      });
      const board = await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      const taskId = await planTaskId(apiClient, seedData.workspaceId, "Phone plan awaiting owner");

      // AC-TASKS-PLAN-BOARD-OPS-002.6: the actions are reachable from the description view.
      await testPage.goto(`/t/${taskId}`);
      const trigger = testPage.getByTestId("plan-decision-trigger");
      await expect(trigger).toBeVisible();
      await expectFullWidthTouchTarget(
        trigger,
        testPage.getByTestId("plan-decision-bar"),
        "decision trigger",
      );
      await trigger.tap();

      const drawer = testPage.getByTestId("plan-decision-drawer");
      await expect(drawer).toBeVisible();
      const returnButton = testPage.getByTestId("plan-decision-drawer-return");
      for (const id of [
        "plan-decision-drawer-accept",
        "plan-decision-drawer-accept-queue",
        "plan-decision-drawer-return",
      ]) {
        await expectFullWidthTouchTarget(testPage.getByTestId(id), drawer, id);
      }
      await expect(returnButton).toBeDisabled();
      await testPage.getByTestId("plan-decision-comment").fill("Split into two plans");
      await expect(returnButton).toBeEnabled();
      await planOpsShot(testPage, "phone-decision-drawer");
      const decided = waitForHttp(testPage, "POST", /\/plan-files\/tasks\/[^/]+\/decision/);
      await returnButton.tap();
      expect((await decided).status()).toBe(200);
      await expect(drawer).toBeHidden();

      // AC-TASKS-PLAN-BOARD-OPS-002.3: the file holds the note, the task is in Queue.
      const written = repo.read(`${PLANS_DIR}/waiting.md`);
      expect(written).toContain("board: queued");
      expect(written).toContain(`- ${today} returned: Split into two plans`);
      expect((await apiClient.getTask(taskId)).workflow_step_id).toBe(board.stepId("Queue"));
      await expect(testPage.getByTestId("plan-decision-bar")).toBeHidden();
    } finally {
      await release();
    }
  });

  test("creates a plan from the full-height drawer opened in the board options", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    try {
      const repo = await seedRepo({
        backend,
        apiClient,
        workspaceId: seedData.workspaceId,
        name: "plan-ops-phone-create",
        files: {
          [`${PLANS_DIR}/seed.md`]: planFileText({
            title: "Phone seed plan",
            board: "queued",
            body: "- [x] Draft\n- [ ] Review",
          }),
        },
      });
      const board = await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      const viewport = testPage.viewportSize();
      if (!viewport) throw new Error("mobile viewport is not set");
      const mobile = new MobileKanbanPage(testPage);
      await testPage.goto(`/?workflowId=${encodeURIComponent(board.boardId)}`);
      await mobile.mobileKanbanLayout().waitFor({ state: "visible" });

      // AC-TASKS-PLAN-BOARD-OPS-005.5: the progress tag renders on the phone card.
      const seedCard = mobile.taskCardByTitle("Phone seed plan");
      await expect(seedCard.getByTestId("kanban-card-progress-chip")).toHaveText("1/2");
      await planOpsShot(testPage, "phone-board");

      // AC-TASKS-PLAN-BOARD-OPS-007.5: the action is in the board display options.
      await mobile.viewOptionsButton.tap();
      await testPage.getByTestId("mobile-display-new-plan").tap();
      const drawer = testPage.getByTestId("new-plan-drawer");
      await expect(drawer).toBeVisible();
      await testPage.getByTestId("new-plan-repository").selectOption({ label: repo.repoName });
      await testPage.getByTestId("new-plan-title").fill("Phone captured idea");

      const drawerBox = await settledBoundingBox(drawer);
      expect(drawerBox.height).toBeGreaterThan(viewport.height * 0.85);
      const scroll = testPage.getByTestId("new-plan-scroll");
      await expect(scroll).toHaveCSS("overflow-y", "auto");
      const create = testPage.getByTestId("new-plan-create");
      await expectFullWidthTouchTarget(create, drawer, "create button");
      const footerBox = await settledBoundingBox(testPage.getByTestId("new-plan-footer"));
      expect(footerBox.y + footerBox.height).toBeLessThanOrEqual(viewport.height + 1);
      await planOpsShot(testPage, "phone-new-plan-drawer");

      const created = waitForHttp(testPage, "POST", /\/plan-files\/plans$/);
      await create.tap();
      expect((await created).status()).toBe(201);
      await expect(drawer).toBeHidden();

      // AC-TASKS-PLAN-BOARD-OPS-007.4: the card is on the board once the action succeeded.
      await expect(mobile.taskCardByTitle("Phone captured idea")).toBeVisible();
      const content = repo.read(`${PLANS_DIR}/phone-captured-idea.md`);
      expect(content).toMatch(/^---\nboard: queued\n/);
      expect(content).toContain("# Phone captured idea\n");
    } finally {
      await release();
    }
  });

  test("opens the waiting list from the navigation sheet with full-width rows", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    try {
      await seedRepo({
        backend,
        apiClient,
        workspaceId: seedData.workspaceId,
        name: "plan-ops-phone-waiting",
        files: {
          [`${PLANS_DIR}/a-dated.md`]: planFileText({
            title: "Phone dated waiting plan",
            board: "waiting_owner",
            date: "2026-01-15",
          }),
          [`${PLANS_DIR}/b-undated.md`]: planFileText({
            title: "Phone undated waiting plan",
            board: "waiting_owner",
          }),
          [`${PLANS_DIR}/c-queued.md`]: planFileText({
            title: "Phone not waiting",
            board: "queued",
          }),
        },
      });
      await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      const datedId = await planTaskId(apiClient, seedData.workspaceId, "Phone dated waiting plan");

      // AC-TASKS-PLAN-BOARD-OPS-009.4: the page is reachable from the navigation sheet.
      const mobile = new MobileKanbanPage(testPage);
      await mobile.goto();
      await mobile.mobileMenuButton.tap();
      await expect(mobile.menuCard).toBeVisible();
      const entry = testPage.getByTestId("mobile-sidebar-plans-waiting");
      await expect(entry).toContainText("2");
      await expectFullWidthTouchTarget(entry, mobile.menuCard, "navigation entry");
      await entry.tap();
      await expect(testPage).toHaveURL(/\/plans\/waiting$/);

      // AC-TASKS-PLAN-BOARD-OPS-009.1 and 009.4: one full-width row of at least 44 px per plan.
      const list = testPage.getByTestId("plans-waiting-list");
      await expect(list).toBeVisible();
      const rows = testPage.locator('[data-testid^="plans-waiting-row-"]');
      await expect(rows).toHaveCount(2);
      await expect(rows.nth(0)).toHaveAttribute("data-testid", `plans-waiting-row-${datedId}`);
      await expect(list).not.toContainText("Phone not waiting");
      for (const index of [0, 1]) {
        await expectFullWidthTouchTarget(rows.nth(index), list, `row ${index}`);
      }
      await planOpsShot(testPage, "phone-waiting-list");

      // AC-TASKS-PLAN-BOARD-OPS-009.2: selecting a row opens the task and its decision trigger.
      await rows.nth(0).tap();
      await expect(testPage).toHaveURL(new RegExp(`/t/${datedId}`));
      await expect(testPage.getByTestId("plan-decision-trigger")).toBeVisible();
    } finally {
      await release();
    }
  });
});
