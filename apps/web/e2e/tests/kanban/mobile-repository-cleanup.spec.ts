import { test, expect } from "../../fixtures/test-base";
import { useDefaultAgentProfile } from "../../helpers/plan-files";
import {
  REPOSITORY_CLEANUP_ENV,
  countCleanupTasks,
  createCleanupRepository,
} from "../../helpers/repository-cleanup";
import { settledBoundingBox } from "../../helpers/settled-box";
import { MobileKanbanPage } from "../../pages/mobile-kanban-page";

const TASK_URL = /\/t\/[^/?#]+/;
const MIN_TOUCH_TARGET = 44;
const HINT = "Choose a repository in the filter to clean it up.";

test.describe("Mobile repository cleanup from the board", () => {
  test.describe.configure({ mode: "serial" });

  test("the listing menu entry cleans the filtered repository in a bottom sheet", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(REPOSITORY_CLEANUP_ENV);
    const repo = await createCleanupRepository({
      backend,
      apiClient,
      workspaceId: seedData.workspaceId,
      name: "e2e-mobile-board-cleanup",
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      const mobile = new MobileKanbanPage(testPage);
      const viewport = testPage.viewportSize();
      if (!viewport) throw new Error("mobile viewport is not set");

      // Two local repositories and no filter: the entry is disabled with the hint.
      await apiClient.saveUserSettings({ workspace_id: seedData.workspaceId, repository_ids: [] });
      await mobile.goto();
      await mobile.viewOptionsButton.tap();
      const entry = testPage.getByTestId("mobile-display-cleanup");
      await expect(entry).toBeDisabled();
      await expect(testPage.locator("#mobile-cleanup-hint")).toHaveText(HINT);
      const disabledBox = await settledBoundingBox(entry);
      expect(disabledBox.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);

      await apiClient.saveUserSettings({
        workspace_id: seedData.workspaceId,
        repository_ids: [repo.repositoryId],
      });
      await mobile.goto();
      await mobile.viewOptionsButton.tap();
      await expect(entry).toBeEnabled();
      await expect(entry).toHaveText("Clean up repository");
      const box = await settledBoundingBox(entry);
      expect(box.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
      expect(box.x + box.width).toBeLessThanOrEqual(viewport.width);
      expect(
        await testPage.evaluate(() => document.documentElement.scrollWidth > window.innerWidth),
      ).toBe(false);

      await entry.tap();
      await expect(mobile.optionsCard).toHaveCount(0);
      const sheet = testPage.getByTestId("repository-cleanup-dialog");
      await expect(sheet).toContainText("Clean up e2e-mobile-board-cleanup?");
      await expect(sheet).toHaveAttribute("data-slot", "drawer-content");
      const sheetBox = await settledBoundingBox(sheet);
      const confirm = await settledBoundingBox(sheet.getByTestId("repository-cleanup-confirm"));
      const cancel = await settledBoundingBox(sheet.getByTestId("repository-cleanup-cancel"));
      expect(sheetBox.y + sheetBox.height).toBeGreaterThan(viewport.height - 4);
      for (const action of [confirm, cancel]) {
        expect(action.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
        expect(action.width).toBeGreaterThanOrEqual(sheetBox.width - 48);
      }

      await sheet.getByTestId("repository-cleanup-cancel").tap();
      await expect(sheet).toBeHidden();
      expect(
        await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-mobile-board-cleanup"),
      ).toBe(0);

      await mobile.viewOptionsButton.tap();
      await entry.tap();
      await testPage.getByTestId("repository-cleanup-confirm").tap();
      await expect(testPage).toHaveURL(TASK_URL);
      expect(
        await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-mobile-board-cleanup"),
      ).toBe(1);
    } finally {
      await apiClient.saveUserSettings({ workspace_id: seedData.workspaceId, repository_ids: [] });
      await repo.cleanup();
      await release();
    }
  });
});
