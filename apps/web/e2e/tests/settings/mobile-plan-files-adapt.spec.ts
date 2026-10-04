import { test, expect } from "../../fixtures/test-base";
import {
  PLAN_FILES_ENV,
  UNADAPTED_FILE_COUNT,
  addRepositoryThroughSettings,
  createUnadaptedRepository,
  useDefaultAgentProfile,
} from "../../helpers/plan-files";
import { settledBoundingBox } from "../../helpers/settled-box";

const TASK_URL = /\/t\/[^/?#]+/;
const MIN_TOUCH_TARGET = 44;

test.describe("Mobile plan file adaptation", () => {
  test.describe.configure({ mode: "serial" });

  test("settings row stacks name, count and a full-width 44px action", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    const repo = await createUnadaptedRepository({
      backend,
      apiClient,
      workspaceId: seedData.workspaceId,
      name: "e2e-mobile-adapt-row",
      register: true,
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/workflows`);
      const row = testPage.locator(`[data-repository-id="${repo.repositoryId}"]`);
      await expect(row).toBeVisible();

      const rowBox = await settledBoundingBox(row);
      const name = await settledBoundingBox(row.getByText("e2e-mobile-adapt-row", { exact: true }));
      const count = await settledBoundingBox(row.getByTestId("plan-files-unadapted-count"));
      const adapt = row.getByTestId("plan-files-adapt");
      const button = await settledBoundingBox(adapt);

      // Name over count over the action, in one column.
      expect(count.y).toBeGreaterThan(name.y);
      expect(button.y).toBeGreaterThan(count.y + count.height - 1);
      expect(button.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
      expect(button.width).toBeGreaterThan(rowBox.width - 32);
      const viewport = testPage.viewportSize();
      if (!viewport) throw new Error("mobile viewport is not set");
      expect(button.x + button.width).toBeLessThanOrEqual(viewport.width);
      expect(
        await testPage.evaluate(() => document.documentElement.scrollWidth > window.innerWidth),
      ).toBe(false);

      await expect(row.getByTestId("plan-files-unadapted-count")).toHaveText(
        `${UNADAPTED_FILE_COUNT} files without board status`,
      );
      await adapt.tap();
      await expect(testPage).toHaveURL(TASK_URL);
    } finally {
      await repo.cleanup();
      await release();
    }
  });

  test("offer is a bottom sheet with stacked full-width 44px actions, primary first", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    const repo = await createUnadaptedRepository({
      backend,
      apiClient,
      workspaceId: seedData.workspaceId,
      name: "e2e-mobile-offer",
      register: false,
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      await addRepositoryThroughSettings(testPage, seedData.workspaceId, repo.repoDir);
      const offer = testPage.getByTestId("adapt-plans-offer");
      await expect(offer).toContainText("Plans found in e2e-mobile-offer");

      const viewport = testPage.viewportSize();
      if (!viewport) throw new Error("mobile viewport is not set");
      const sheet = await settledBoundingBox(offer);
      const confirm = await settledBoundingBox(offer.getByTestId("adapt-plans-confirm"));
      const notNow = await settledBoundingBox(offer.getByTestId("adapt-plans-not-now"));

      expect(sheet.x + sheet.width).toBeLessThanOrEqual(viewport.width);
      expect(sheet.y + sheet.height).toBeGreaterThan(viewport.height - 4);
      expect(sheet.y).toBeGreaterThanOrEqual(0);
      await expect(offer).toHaveAttribute("data-slot", "drawer-content");
      for (const box of [confirm, notNow]) {
        expect(box.height).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET);
        // The drawer's own inset (8px) plus the footer padding (16px) on each side.
        expect(box.width).toBeGreaterThanOrEqual(sheet.width - 48);
      }
      expect(confirm.y + confirm.height).toBeLessThanOrEqual(notNow.y + 1);
      expect(
        await testPage.evaluate(() => document.documentElement.scrollWidth > window.innerWidth),
      ).toBe(false);

      await offer.getByTestId("adapt-plans-not-now").tap();
      await expect(offer).toBeHidden();
    } finally {
      await repo.cleanup();
      await release();
    }
  });
});
