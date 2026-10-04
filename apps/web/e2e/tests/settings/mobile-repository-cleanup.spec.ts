import { test, expect } from "../../fixtures/test-base";
import { useDefaultAgentProfile } from "../../helpers/plan-files";
import {
  REPOSITORY_CLEANUP_ENV,
  countCleanupTasks,
  createCleanupRepository,
} from "../../helpers/repository-cleanup";
import { settledBoundingBox } from "../../helpers/settled-box";

const TASK_URL = /\/t\/[^/?#]+/;
const MIN_TOUCH_TARGET = 44;

test.describe("Mobile repository cleanup", () => {
  test.describe.configure({ mode: "serial" });

  test("broom is a 44px icon button below the name and the sheet stacks full-width actions", async ({
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
      name: "e2e-mobile-cleanup",
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/repositories`);
      const row = testPage.locator('[data-slot="card"]').filter({ hasText: "e2e-mobile-cleanup" });
      const broom = row.getByTestId("repository-cleanup-button");
      await expect(broom).toBeVisible();

      const viewport = testPage.viewportSize();
      if (!viewport) throw new Error("mobile viewport is not set");
      const name = await settledBoundingBox(row.getByText("e2e-mobile-cleanup", { exact: true }));
      const broomBox = await settledBoundingBox(broom);
      const edit = await settledBoundingBox(row.getByRole("button", { name: "Edit" }));
      expect(broomBox.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
      expect(broomBox.width).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
      expect(broomBox.y).toBeGreaterThan(name.y + name.height - 1);
      expect(broomBox.x).toBeLessThan(edit.x);
      expect(edit.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
      expect(broomBox.x + broomBox.width).toBeLessThanOrEqual(viewport.width);
      expect(
        await testPage.evaluate(() => document.documentElement.scrollWidth > window.innerWidth),
      ).toBe(false);

      await broom.tap();
      const sheet = testPage.getByTestId("repository-cleanup-dialog");
      await expect(sheet).toContainText("Clean up e2e-mobile-cleanup?");
      await expect(sheet).toHaveAttribute("data-slot", "drawer-content");
      const sheetBox = await settledBoundingBox(sheet);
      const confirm = await settledBoundingBox(sheet.getByTestId("repository-cleanup-confirm"));
      const cancel = await settledBoundingBox(sheet.getByTestId("repository-cleanup-cancel"));
      expect(sheetBox.y + sheetBox.height).toBeGreaterThan(viewport.height - 4);
      for (const box of [confirm, cancel]) {
        expect(box.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
        // The drawer's own inset (8px) plus the footer padding (16px) on each side.
        expect(box.width).toBeGreaterThanOrEqual(sheetBox.width - 48);
      }
      expect(confirm.y + confirm.height).toBeLessThanOrEqual(cancel.y + 1);

      await sheet.getByTestId("repository-cleanup-cancel").tap();
      await expect(sheet).toBeHidden();
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-mobile-cleanup")).toBe(
        0,
      );

      await broom.tap();
      await testPage.getByTestId("repository-cleanup-confirm").tap();
      await expect(testPage).toHaveURL(TASK_URL);
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-mobile-cleanup")).toBe(
        1,
      );
    } finally {
      await repo.cleanup();
      await release();
    }
  });
});
