import { test, expect } from "../../fixtures/test-base";
import {
  PLAN_FILES_ENV,
  UNADAPTED_FILE_COUNT,
  addRepositoryThroughSettings,
  createUnadaptedRepository,
  useDefaultAgentProfile,
} from "../../helpers/plan-files";

const TASK_URL = /\/t\/[^/?#]+/;
const BOARD_NOTE = "A Plans board will be created and sync turned on.";
const EXISTING_NOTE = "A task for this is already running. Opening it.";

function taskId(url: string): string {
  const match = TASK_URL.exec(url);
  if (!match) throw new Error(`not a task url: ${url}`);
  return match[0];
}

test.describe("Plan file adaptation", () => {
  test.describe.configure({ mode: "serial" });

  test("settings row starts one adaptation task and a second start opens the same task", async ({
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
      name: "e2e-adapt-row",
      register: true,
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      const workflows = `/settings/workspaces/${seedData.workspaceId}/workflows`;
      await testPage.goto(workflows);

      const row = testPage.locator(`[data-repository-id="${repo.repositoryId}"]`);
      await expect(row).toContainText("e2e-adapt-row");
      await expect(row.getByTestId("plan-files-unadapted-count")).toHaveText(
        `${UNADAPTED_FILE_COUNT} files without board status`,
      );
      await expect(row.getByText(BOARD_NOTE)).toBeVisible();
      const adapt = row.getByTestId("plan-files-adapt");
      const box = await adapt.boundingBox();
      expect(Math.round(box?.height ?? 0)).toBe(28);

      await adapt.click();
      await expect(testPage).toHaveURL(TASK_URL);
      const firstTask = taskId(testPage.url());

      // Starting created the board and turned sync on, so the helper line is gone.
      await testPage.goto(workflows);
      await expect(testPage.getByTestId("plan-files-enabled")).toBeChecked();
      const again = testPage.locator(`[data-repository-id="${repo.repositoryId}"]`);
      await expect(again).toBeVisible();
      await expect(again.getByText(BOARD_NOTE)).toHaveCount(0);

      await again.getByTestId("plan-files-adapt").click();
      await expect(testPage).toHaveURL(TASK_URL);
      expect(taskId(testPage.url())).toBe(firstTask);
      await expect(testPage.getByText(EXISTING_NOTE)).toBeVisible();
    } finally {
      await repo.cleanup();
      await release();
    }
  });

  test("offer after adding a repository: Not now has no side effects", async ({
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
      name: "e2e-offer-skip",
      register: false,
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      await addRepositoryThroughSettings(testPage, seedData.workspaceId, repo.repoDir);
      const offer = testPage.getByTestId("adapt-plans-offer");
      await expect(offer).toContainText("Plans found in e2e-offer-skip");
      await expect(offer).toContainText(
        `${UNADAPTED_FILE_COUNT} plan files are not in the board format`,
      );
      await expect(offer.getByText(BOARD_NOTE)).toBeVisible();

      await offer.getByTestId("adapt-plans-not-now").click();
      await expect(offer).toBeHidden();
      await expect(testPage).not.toHaveURL(TASK_URL);
      const tasks = await apiClient.listTasks(seedData.workspaceId);
      expect(tasks.tasks.some((task) => task.title.startsWith("Adapt plan files"))).toBe(false);
    } finally {
      await repo.cleanup();
      await release();
    }
  });

  test("offer after adding a repository: Adapt with agent opens the created task", async ({
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
      name: "e2e-offer-adapt",
      register: false,
    });
    try {
      await useDefaultAgentProfile(apiClient, seedData.workspaceId, seedData.agentProfileId);
      await addRepositoryThroughSettings(testPage, seedData.workspaceId, repo.repoDir);
      const offer = testPage.getByTestId("adapt-plans-offer");
      await expect(offer).toContainText("Plans found in e2e-offer-adapt");
      await offer.getByTestId("adapt-plans-confirm").click();
      await expect(testPage).toHaveURL(TASK_URL);
      const tasks = await apiClient.listTasks(seedData.workspaceId);
      expect(tasks.tasks.some((task) => task.title.includes("e2e-offer-adapt"))).toBe(true);
    } finally {
      await repo.cleanup();
      await release();
    }
  });
});
