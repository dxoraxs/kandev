import { test, expect } from "../../fixtures/test-base";
import { useDefaultAgentProfile } from "../../helpers/plan-files";
import {
  REPOSITORY_CLEANUP_ENV,
  countCleanupTasks,
  createCleanupRepository,
} from "../../helpers/repository-cleanup";

const TASK_URL = /\/t\/[^/?#]+/;
const EXISTING_NOTE = "A task for this is already running. Opening it.";
const HINT = "Choose a repository in the filter to clean it up.";
const BROOM = "board-cleanup-button";

function taskId(url: string): string {
  const match = TASK_URL.exec(url);
  if (!match) throw new Error(`not a task url: ${url}`);
  return match[0];
}

test.describe("Repository cleanup from the board", () => {
  test.describe.configure({ mode: "serial" });

  test("the broom cleans the repository selected in the board filter", async ({
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
      name: "e2e-board-cleanup",
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);

      // Two local repositories and no filter: the broom asks for a choice.
      await apiClient.saveUserSettings({ workspace_id: seedData.workspaceId, repository_ids: [] });
      await testPage.goto("/");
      const broom = testPage.getByTestId(BROOM);
      await expect(broom).toBeVisible();
      await expect(broom).toHaveAttribute("aria-disabled", "true");
      await broom.hover();
      await expect(testPage.getByRole("tooltip", { name: HINT })).toBeVisible();
      // Playwright treats aria-disabled as not actionable; a pointer still reaches it.
      await broom.click({ force: true });
      await expect(testPage.getByTestId("toast-message").filter({ hasText: HINT })).toBeVisible();
      await expect(testPage.getByTestId("repository-cleanup-dialog")).toHaveCount(0);

      await apiClient.saveUserSettings({
        workspace_id: seedData.workspaceId,
        repository_ids: [repo.repositoryId],
      });
      await testPage.goto("/");
      await expect(broom).not.toHaveAttribute("aria-disabled", "true");
      await expect(broom).toHaveAccessibleName("Clean up repository");
      await broom.hover();
      await expect(testPage.getByRole("tooltip", { name: "Clean up repository" })).toBeVisible();

      await broom.click();
      const dialog = testPage.getByTestId("repository-cleanup-dialog");
      await expect(dialog).toContainText("Clean up e2e-board-cleanup?");
      await expect(dialog.locator("ol > li")).toHaveCount(4);
      await dialog.getByTestId("repository-cleanup-cancel").click();
      await expect(dialog).toBeHidden();
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-board-cleanup")).toBe(0);

      await broom.click();
      await testPage.getByTestId("repository-cleanup-confirm").click();
      await expect(testPage).toHaveURL(TASK_URL);
      const firstTask = taskId(testPage.url());
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-board-cleanup")).toBe(1);

      await testPage.goto("/");
      await testPage.getByTestId(BROOM).click();
      await testPage.getByTestId("repository-cleanup-confirm").click();
      await expect(testPage).toHaveURL(TASK_URL);
      expect(taskId(testPage.url())).toBe(firstTask);
      await expect(testPage.getByText(EXISTING_NOTE)).toBeVisible();
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-board-cleanup")).toBe(1);

      // The action moved: repository settings rows no longer carry it.
      await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/repositories`);
      const row = testPage.locator('[data-slot="card"]').filter({ hasText: "e2e-board-cleanup" });
      await expect(row.getByRole("button", { name: "Edit" })).toBeVisible();
      await expect(row.getByRole("button", { name: "Clean up repository" })).toHaveCount(0);
    } finally {
      await apiClient.saveUserSettings({ workspace_id: seedData.workspaceId, repository_ids: [] });
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
      name: "e2e-board-noprofile",
    });
    try {
      await apiClient.updateWorkspace(seedData.workspaceId, { default_agent_profile_id: "" });
      await apiClient.saveUserSettings({
        workspace_id: seedData.workspaceId,
        repository_ids: [repo.repositoryId],
      });
      await testPage.goto("/");
      await testPage.getByTestId(BROOM).click();
      const dialog = testPage.getByTestId("repository-cleanup-dialog");
      await dialog.getByTestId("repository-cleanup-confirm").click();
      await expect(
        testPage.getByText("Set a default agent profile for this workspace first."),
      ).toBeVisible();
      await expect(dialog).toBeVisible();
      await expect(testPage).not.toHaveURL(TASK_URL);
      expect(await countCleanupTasks(apiClient, seedData.workspaceId, "e2e-board-noprofile")).toBe(
        0,
      );
    } finally {
      await apiClient.saveUserSettings({ workspace_id: seedData.workspaceId, repository_ids: [] });
      await repo.cleanup();
      await release();
    }
  });

  test("the broom is absent while the runtime flag is off", async ({ testPage }) => {
    await testPage.goto("/");
    await expect(testPage.getByTestId("view-toggle-kanban")).toBeVisible();
    await expect(testPage.getByTestId(BROOM)).toHaveCount(0);
  });
});
