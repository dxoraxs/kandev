import { test, expect } from "../../fixtures/test-base";
import { KanbanPage } from "../../pages/kanban-page";
import { waitForHttp } from "../../helpers/causal-waits";
import {
  PLAN_FILES_ENV,
  configurePlanBoard,
  createPlanFilesRepository,
  type PlanFilesRepository,
  dragCardToColumn,
  localDay,
  planFileText,
  planOpsShot,
  planTaskId,
  syncPlanFiles,
} from "../../helpers/plan-files";

const PLANS_DIR = "docs/plans";

const seeded: PlanFilesRepository[] = [];

/** Creates a plan repository that the test unregisters when it ends. */
async function seedRepo(options: Parameters<typeof createPlanFilesRepository>[0]) {
  const repo = await createPlanFilesRepository(options);
  seeded.push(repo);
  return repo;
}

test.describe("Plan board operations", () => {
  test.describe.configure({ mode: "serial" });

  test.afterEach(async () => {
    for (const repo of seeded.splice(0)) await repo.cleanup();
  });

  test("returns a waiting plan with a comment and records the note and the queued status", async ({
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
        name: "plan-ops-decision",
        files: {
          [`${PLANS_DIR}/waiting.md`]: planFileText({
            title: "Plan awaiting owner",
            board: "waiting_owner",
            date: today,
          }),
          [`${PLANS_DIR}/queued.md`]: planFileText({ title: "Plan still queued", board: "queued" }),
        },
      });
      const board = await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      const waitingId = await planTaskId(apiClient, seedData.workspaceId, "Plan awaiting owner");

      // AC-TASKS-PLAN-BOARD-OPS-002.1: the waiting plan offers the decision.
      await testPage.goto(`/t/${waitingId}`);
      const bar = testPage.getByTestId("plan-decision-bar");
      await expect(bar).toBeVisible();
      await planOpsShot(testPage, "desktop-decision-bar");

      // AC-TASKS-PLAN-BOARD-OPS-002.3: Return needs a non-empty comment.
      await testPage.getByTestId("plan-decision-return").click();
      const dialog = testPage.getByTestId("plan-decision-dialog");
      await expect(dialog).toBeVisible();
      const confirm = testPage.getByTestId("plan-decision-confirm");
      await expect(confirm).toBeDisabled();
      await testPage.getByTestId("plan-decision-comment").fill("Needs a rollout section");
      await expect(confirm).toBeEnabled();
      const decided = waitForHttp(testPage, "POST", /\/plan-files\/tasks\/[^/]+\/decision/);
      await confirm.click();
      expect((await decided).status()).toBe(200);
      await expect(dialog).toBeHidden();

      // AC-TASKS-PLAN-BOARD-OPS-002.6 (desktop bar) and 002.3: the file holds the
      // decision, and the task is in the Queue step before the response returned.
      const written = repo.read(`${PLANS_DIR}/waiting.md`);
      expect(written).toContain("board: queued");
      expect(written).toContain(`- ${today} returned: Needs a rollout section`);
      expect(written).toContain("## Owner notes");
      expect((await apiClient.getTask(waitingId)).workflow_step_id).toBe(board.stepId("Queue"));
      await expect(bar).toBeHidden();

      // The queued plan never shows the decision.
      const queuedId = await planTaskId(apiClient, seedData.workspaceId, "Plan still queued");
      expect((await apiClient.getTask(queuedId)).workflow_step_id).toBe(board.stepId("Queue"));
    } finally {
      await release();
    }
  });

  test("creates a plan from the board and shows its card and file", async ({
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
        name: "plan-ops-create",
        files: {
          [`${PLANS_DIR}/seed.md`]: planFileText({ title: "Seed plan", board: "queued" }),
        },
      });
      const board = await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      await testPage.setViewportSize({ width: 2400, height: 900 });
      const kanban = new KanbanPage(testPage);
      await kanban.goto(board.boardId);

      // AC-TASKS-PLAN-BOARD-OPS-007.1: the plan board offers New plan.
      await testPage.getByTestId("new-plan-button").click();
      const dialog = testPage.getByTestId("new-plan-dialog");
      await expect(dialog).toBeVisible();
      await testPage.getByTestId("new-plan-repository").selectOption({ label: repo.repoName });
      await expect(testPage.getByTestId("new-plan-directory")).toHaveValue(PLANS_DIR);
      const create = testPage.getByTestId("new-plan-create");
      await expect(create).toBeDisabled();
      await testPage.getByTestId("new-plan-title").fill("Captured idea");
      await testPage.getByTestId("new-plan-priority").selectOption("high");
      await testPage.getByTestId("new-plan-body").fill("Why this matters.");
      await expect(create).toBeEnabled();
      await planOpsShot(testPage, "desktop-new-plan-dialog");
      const created = waitForHttp(testPage, "POST", /\/plan-files\/plans$/);
      await create.click();
      expect((await created).status()).toBe(201);
      await expect(dialog).toBeHidden();

      // AC-TASKS-PLAN-BOARD-OPS-007.4: the card is on the board once the action succeeded.
      await expect(kanban.taskCardInColumn("Captured idea", board.stepId("Queue"))).toBeVisible();

      // AC-TASKS-PLAN-BOARD-OPS-007.2 and 007.3: slug name, board header, heading, body.
      const content = repo.read(`${PLANS_DIR}/captured-idea.md`);
      expect(content).toMatch(/^---\nboard: queued\n/);
      expect(content).toContain('title: "Captured idea"');
      expect(content).toContain("priority: high");
      expect(content).toContain("# Captured idea\n");
      expect(content).toContain("Why this matters.");

      // AC-TASKS-PLAN-BOARD-OPS-007.3: an existing file is never replaced.
      await testPage.getByTestId("new-plan-button").click();
      await testPage.getByTestId("new-plan-repository").selectOption({ label: repo.repoName });
      await testPage.getByTestId("new-plan-title").fill("Captured idea");
      const rejected = waitForHttp(testPage, "POST", /\/plan-files\/plans$/);
      await testPage.getByTestId("new-plan-create").click();
      expect((await rejected).status()).toBe(409);
      await expect(testPage.getByTestId("new-plan-error")).toBeVisible();
      expect(repo.read(`${PLANS_DIR}/captured-idea.md`)).toBe(content);
      await testPage.getByTestId("new-plan-cancel").click();
      await expect(testPage.getByTestId("new-plan-dialog")).toBeHidden();
    } finally {
      await release();
    }
  });

  test("flags a board edit as uncommitted and clears it with Commit plan files", async ({
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
        name: "plan-ops-commit",
        files: {
          [`${PLANS_DIR}/moving.md`]: planFileText({
            title: "Moving plan",
            board: "queued",
            body: "- [x] Draft\n- [ ] Review\n- [ ] Ship",
          }),
          [`${PLANS_DIR}/finished.md`]: planFileText({
            title: "Finished plan",
            board: "done",
            body: "- [x] Built\n- [ ] Loose end",
          }),
        },
      });
      const board = await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      // Wide enough that every column is on screen, so a drag never scrolls the board.
      await testPage.setViewportSize({ width: 2400, height: 900 });
      const kanban = new KanbanPage(testPage);
      await kanban.goto(board.boardId);

      // AC-TASKS-PLAN-BOARD-OPS-005.5: progress and flag tags on the cards.
      const queued = kanban.taskCardInColumn("Moving plan", board.stepId("Queue"));
      await expect(queued.getByTestId("kanban-card-progress-chip")).toHaveText("1/3");
      await expect(
        kanban
          .taskCardInColumn("Finished plan", board.stepId("Done"))
          .getByTestId("kanban-card-flag-open_items"),
      ).toBeVisible();
      await expect(queued.getByTestId("kanban-card-flag-uncommitted")).toHaveCount(0);

      // A board move is the edit that leaves the file uncommitted.
      await dragCardToColumn(testPage, queued, kanban.columnByStepId(board.stepId("In progress")));
      const moved = kanban.taskCardInColumn("Moving plan", board.stepId("In progress"));
      await expect(moved).toBeVisible();
      await expect.poll(() => repo.read(`${PLANS_DIR}/moving.md`)).toContain("board: in_progress");
      await syncPlanFiles(apiClient, seedData.workspaceId);
      await expect(moved.getByTestId("kanban-card-flag-uncommitted")).toBeVisible();
      await planOpsShot(testPage, "desktop-board-flags");

      // AC-TASKS-PLAN-BOARD-OPS-008.2: the settings list the file and offer the commit.
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/workflows`);
      const row = testPage.getByTestId("plan-files-git-row").filter({ hasText: repo.repoName });
      await expect(row).toHaveCount(1);
      await expect(row.getByTestId("plan-files-git-count")).toHaveText("1 file");
      await testPage.getByTestId("plan-files-operations").scrollIntoViewIfNeeded();
      await planOpsShot(testPage, "desktop-settings-operations");
      await row.scrollIntoViewIfNeeded();
      await planOpsShot(testPage, "desktop-settings-git");
      await row.getByTestId("plan-files-git-commit").click();
      const dialog = testPage.getByTestId("plan-files-commit-dialog");
      await expect(dialog).toBeVisible();
      await expect(testPage.getByTestId("plan-files-commit-files")).toContainText("moving.md");
      await expect(testPage.getByTestId("plan-files-commit-message")).toHaveValue(
        "docs(plans): update plan files",
      );
      await planOpsShot(testPage, "desktop-commit-dialog");
      const committed = waitForHttp(testPage, "POST", /\/plan-files\/commit$/);
      await testPage.getByTestId("plan-files-commit-confirm").click();
      expect((await committed).status()).toBe(200);
      await expect(dialog).toBeHidden();

      // AC-TASKS-PLAN-BOARD-OPS-008.3: one commit holds exactly the plan file; nothing is pushed.
      expect(repo.git("log -1 --format=%s")).toBe("docs(plans): update plan files");
      expect(repo.git("show --name-only --format= HEAD")).toBe(`${PLANS_DIR}/moving.md`);
      expect(repo.git("status --porcelain")).toBe("");
      await expect(row.getByTestId("plan-files-git-count")).toHaveText("0 files");
      await expect(row.getByTestId("plan-files-git-commit")).toHaveCount(0);

      // AC-TASKS-PLAN-BOARD-OPS-008.1: the next pass clears the flag.
      await syncPlanFiles(apiClient, seedData.workspaceId);
      await kanban.goto(board.boardId);
      await expect(
        kanban.taskCardInColumn("Moving plan", board.stepId("In progress")),
      ).toBeVisible();
      await expect(
        kanban
          .taskCardInColumn("Moving plan", board.stepId("In progress"))
          .getByTestId("kanban-card-flag-uncommitted"),
      ).toHaveCount(0);
    } finally {
      await release();
    }
  });

  test("lists waiting plans, dated first, and opens one with the decision bar", async ({
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
        name: "plan-ops-waiting",
        files: {
          [`${PLANS_DIR}/a-undated.md`]: planFileText({
            title: "Undated waiting plan",
            board: "waiting_owner",
          }),
          [`${PLANS_DIR}/b-dated.md`]: planFileText({
            title: "Dated waiting plan",
            board: "waiting_owner",
            date: "2026-01-15",
          }),
          [`${PLANS_DIR}/c-queued.md`]: planFileText({ title: "Not waiting", board: "queued" }),
        },
      });
      await configurePlanBoard(testPage, apiClient, seedData.workspaceId);
      const datedId = await planTaskId(apiClient, seedData.workspaceId, "Dated waiting plan");
      const undatedId = await planTaskId(apiClient, seedData.workspaceId, "Undated waiting plan");

      // AC-TASKS-PLAN-BOARD-OPS-009.2: the navigation entry counts the listed plans.
      await testPage.goto("/");
      await expect(testPage.getByTestId("sidebar-plans-waiting")).toContainText("2");

      // AC-TASKS-PLAN-BOARD-OPS-009.1: dated plans first, undated last, no queued plan.
      await testPage.getByTestId("sidebar-plans-waiting").click();
      await expect(testPage.getByTestId("plans-waiting-page")).toBeVisible();
      await expect(testPage).toHaveURL(/\/plans\/waiting$/);
      const rows = testPage.locator('[data-testid^="plans-waiting-row-"]');
      await expect(rows).toHaveCount(2);
      await expect(rows.nth(0)).toHaveAttribute("data-testid", `plans-waiting-row-${datedId}`);
      await expect(rows.nth(1)).toHaveAttribute("data-testid", `plans-waiting-row-${undatedId}`);
      await expect(rows.nth(0)).toContainText("plan-ops-waiting");
      await expect(testPage.getByTestId("plans-waiting-page")).not.toContainText("Not waiting");
      await planOpsShot(testPage, "desktop-waiting-owner");

      // AC-TASKS-PLAN-BOARD-OPS-009.2: selecting a row opens the task and its decision bar.
      await rows.nth(0).click();
      await expect(testPage).toHaveURL(new RegExp(`/t/${datedId}`));
      await expect(testPage.getByTestId("plan-decision-bar")).toBeVisible();
    } finally {
      await release();
    }
  });
});
