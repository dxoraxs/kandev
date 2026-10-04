import { test, expect } from "../../fixtures/test-base";

// The testPage fixture clears the workspace default agent profile before every
// test, so a task created without agent_profile_id resolves no profile on open.
test.describe("Mobile task open without an agent profile", () => {
  // @covers AC-TASKS-TASK-DESCRIPTION-VIEW-004.1, 004.2
  test("shows the description first and the notice as a touch drawer", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    test.setTimeout(120_000);
    await testPage.setViewportSize({ width: 390, height: 844 });
    const task = await apiClient.createTask(seedData.workspaceId, "Mobile unassigned task", {
      description: "## Now\n\n**Next responsible:** owner reviews the deletion flow.",
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      repository_ids: [seedData.repositoryId],
    });

    await testPage.goto(`/t/${task.id}`);

    const view = testPage.getByTestId("sessionless-task-view");
    await expect(view).toHaveCount(1);
    await expect(view.getByTestId("sessionless-task-description").locator("h2")).toHaveText("Now");
    const nav = testPage.getByTestId("session-mobile-bottom-nav");
    await expect(nav.getByRole("button", { name: "Description" })).toBeVisible();
    await expect(nav.getByRole("button", { name: "Chat" })).toHaveCount(0);
    await expect(testPage.getByTestId("mobile-sessions-pill")).toHaveCount(0);
    await expect(testPage.getByTestId("chat-input-area")).toHaveCount(0);
    await expect(testPage.getByTestId("ensure-session-error-banner")).toHaveCount(0);

    const control = testPage.getByTestId("sessionless-unassigned-notice");
    await expect(control).toHaveCount(1);
    expect((await control.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    const overflow = await testPage.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);

    await control.tap();
    const drawer = testPage.getByTestId("sessionless-notice-drawer");
    await expect(drawer).toBeVisible();
    for (const testId of ["sessionless-start-agent", "sessionless-workspace-settings"]) {
      const box = await drawer.getByTestId(testId).boundingBox();
      expect(box, testId).not.toBeNull();
      expect(box!.height, testId).toBeGreaterThanOrEqual(44);
    }

    await drawer.getByTestId("sessionless-start-agent").tap();
    await expect(testPage.getByRole("dialog").filter({ hasText: "New agent in" })).toBeVisible();
  });
});
