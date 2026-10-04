import { test, expect } from "../../fixtures/test-base";
import { PLAN_FILES_ENV, createPlanRepository } from "../../helpers/plan-files";

test.describe("Mobile plan files settings", () => {
  test("shows full-width controls and Sync now updates the status line", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const release = await backend.useEnv(PLAN_FILES_ENV);
    try {
      await createPlanRepository({
        backend,
        apiClient,
        workspaceId: seedData.workspaceId,
        title: "Plan files mobile",
      });
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/workflows`);
      const section = testPage.getByTestId("plan-files-section");
      await expect(section).toBeVisible();
      await testPage.getByTestId("plan-files-enabled").click();
      await testPage.getByTestId("plan-files-create-board").click();

      const viewport = testPage.viewportSize();
      if (!viewport) throw new Error("mobile viewport is not set");
      const sectionBox = await section.boundingBox();
      if (!sectionBox) throw new Error("plan files section has no layout box");
      expect(sectionBox.x + sectionBox.width).toBeLessThanOrEqual(viewport.width + 1);

      const save = testPage.getByTestId("plan-files-save");
      await expect(save).toBeEnabled();
      await save.click();
      const syncNow = testPage.getByTestId("plan-files-sync-now");
      await expect(syncNow).toBeEnabled();

      // Controls stack to the full width of the card on a phone.
      const card = (await section.locator("> div").first().boundingBox())!;
      for (const control of [
        syncNow,
        save,
        testPage.getByTestId("plan-files-create-board"),
        testPage.getByTestId("plan-files-board"),
      ]) {
        const box = await control.boundingBox();
        if (!box) throw new Error("control has no layout box");
        expect(box.width).toBeGreaterThan(card.width - 48);
      }

      const status = testPage.getByTestId("plan-files-status");
      await expect(status).toHaveAttribute("data-state", "waiting");
      await syncNow.tap();
      await expect(status).toHaveAttribute("data-state", "ok");
      await expect(testPage.getByTestId("plan-files-counts")).toContainText("1 created");
    } finally {
      await release();
    }
  });
});
