import { test, expect } from "../../fixtures/test-base";
import { useDefaultAgentProfile } from "../../helpers/plan-files";
import {
  REPOSITORY_CLEANUP_ENV,
  countCleanupTasks,
  createCleanupRepository,
} from "../../helpers/repository-cleanup";

const TASK_URL = /\/t\/[^/?#]+/;
const EXISTING_NOTE = "A task for this is already running. Opening it.";
const BROOM = "repository-cleanup-button";

function taskId(url: string): string {
  const match = TASK_URL.exec(url);
  if (!match) throw new Error(`not a task url: ${url}`);
  return match[0];
}

test.describe("Repository cleanup", () => {
  test.describe.configure({ mode: "serial" });

  test("broom confirms, opens one cleanup task, and a second start opens the same task", async ({
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
      name: "e2e-cleanup-row",
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      const repositories = `/settings/workspaces/${seedData.workspaceId}/repositories`;
      await testPage.goto(repositories);

      const row = testPage.locator('[data-slot="card"]').filter({ hasText: "e2e-cleanup-row" });
      const broom = row.getByTestId(BROOM);
      await expect(broom).toBeVisible();
      await expect(broom).toHaveAccessibleName("Clean up repository");
      const box = await broom.boundingBox();
      expect(Math.round(box?.height ?? 0)).toBe(28);
      expect(Math.round(box?.width ?? 0)).toBe(28);

      await broom.hover();
      await expect(testPage.getByRole("tooltip", { name: "Clean up repository" })).toBeVisible();

      // Cancel creates nothing and does not open the repository editor.
      await broom.click();
      const dialog = testPage.getByTestId("repository-cleanup-dialog");
      await expect(dialog).toContainText("Clean up e2e-cleanup-row?");
      await expect(dialog.locator("ol > li")).toHaveCount(4);
      await dialog.getByTestId("repository-cleanup-cancel").click();
      await expect(dialog).toBeHidden();
      await expect(testPage).not.toHaveURL(TASK_URL);
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-cleanup-row")).toBe(0);

      await broom.click();
      await testPage.getByTestId("repository-cleanup-confirm").click();
      await expect(testPage).toHaveURL(TASK_URL);
      const firstTask = taskId(testPage.url());
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-cleanup-row")).toBe(1);

      await testPage.goto(repositories);
      const again = testPage.locator('[data-slot="card"]').filter({ hasText: "e2e-cleanup-row" });
      await again.getByTestId(BROOM).click();
      await testPage.getByTestId("repository-cleanup-confirm").click();
      await expect(testPage).toHaveURL(TASK_URL);
      expect(taskId(testPage.url())).toBe(firstTask);
      await expect(testPage.getByText(EXISTING_NOTE)).toBeVisible();
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-cleanup-row")).toBe(1);
    } finally {
      await repo.cleanup();
      await release();
    }
  });

  test("a missing default agent profile keeps the dialog open and creates no task", async ({
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
      name: "e2e-cleanup-noprofile",
    });
    try {
      await apiClient.updateWorkspace(seedData.workspaceId, { default_agent_profile_id: "" });
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/repositories`);
      const row = testPage
        .locator('[data-slot="card"]')
        .filter({ hasText: "e2e-cleanup-noprofile" });
      await row.getByTestId(BROOM).click();
      const dialog = testPage.getByTestId("repository-cleanup-dialog");
      await dialog.getByTestId("repository-cleanup-confirm").click();
      await expect(
        testPage.getByText("Set a default agent profile for this workspace first."),
      ).toBeVisible();
      await expect(dialog).toBeVisible();
      await expect(testPage).not.toHaveURL(TASK_URL);
      expect(
        await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-cleanup-noprofile"),
      ).toBe(0);
    } finally {
      await repo.cleanup();
      await release();
    }
  });

  test("the broom is absent while the runtime flag is off", async ({
    testPage,
    backend,
    apiClient,
    seedData,
  }) => {
    const repo = await createCleanupRepository({
      backend,
      apiClient,
      workspaceId: seedData.workspaceId,
      name: "e2e-cleanup-flag-off",
    });
    try {
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/repositories`);
      const row = testPage
        .locator('[data-slot="card"]')
        .filter({ hasText: "e2e-cleanup-flag-off" });
      await expect(row.getByRole("button", { name: "Edit" })).toBeVisible();
      await expect(row.getByTestId(BROOM)).toHaveCount(0);
    } finally {
      await repo.cleanup();
    }
  });
});
