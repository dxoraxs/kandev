import type { Locator, Page } from "@playwright/test";
import { test, expect } from "../../fixtures/test-base";
import { SessionPage } from "../../pages/session-page";

const DESCRIPTION = [
  "## Waiting on the owner",
  "",
  "**Next responsible:** owner deploys the migration.",
  "",
  "- deploy the migration",
  "- confirm the board moves",
].join("\n");
const LONG_DESCRIPTION = Array.from(
  { length: 60 },
  (_, i) => `Paragraph ${i + 1} of the plan body.`,
).join("\n\n");

function descriptionTab(page: Page): Locator {
  return page.locator(".dv-tab").filter({ hasText: /^Description$/ });
}

function sessionTabs(page: Page): Locator {
  return page.locator(".dv-tab").filter({ has: page.locator('[data-testid^="session-tab-"]') });
}

/** The floated notice never covers a paragraph of the description it sits in. */
async function expectNoticeBesideText(page: Page): Promise<void> {
  const notice = await page.getByTestId("sessionless-unassigned-notice").boundingBox();
  expect(notice).not.toBeNull();
  const blocks = page.getByTestId("sessionless-task-description").locator("p, h2, li");
  // Line boxes, not block boxes: a block extends under a float, its text never does.
  const lineBoxes = await blocks.evaluateAll((els) =>
    els.flatMap((el) => {
      const range = document.createRange();
      range.selectNodeContents(el);
      return Array.from(range.getClientRects()).map((r) => r.toJSON() as DOMRect);
    }),
  );
  expect(lineBoxes.length).toBeGreaterThan(0);
  for (const box of lineBoxes) {
    const overlaps =
      box.x < notice!.x + notice!.width &&
      notice!.x < box.x + box.width &&
      box.y < notice!.y + notice!.height &&
      notice!.y < box.y + box.height;
    expect(overlaps, JSON.stringify(box)).toBe(false);
  }
}

// The testPage fixture clears the workspace default agent profile before every
// test, and the seeded workflow steps carry no profile, so a task created
// without agent_profile_id resolves no profile on open.
test.describe("Task open without an agent profile", () => {
  // @covers AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.1, 003.2
  // @covers AC-TASKS-TASK-DESCRIPTION-VIEW-001.1, 002.1, 002.2, 003.1, 003.2
  test("opens on a Description tab with a corner notice, then adds an Agent tab", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    test.setTimeout(120_000);
    const task = await apiClient.createTask(seedData.workspaceId, "Unassigned plan task", {
      description: DESCRIPTION,
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      repository_ids: [seedData.repositoryId],
    });

    await testPage.goto(`/t/${task.id}`);

    const view = testPage.getByTestId("sessionless-task-view");
    await expect(view).toHaveCount(1);
    const document = view.getByTestId("sessionless-task-description");
    await expect(document.locator("h2")).toHaveText("Waiting on the owner");
    await expect(document.locator("li")).toHaveCount(2);
    await expect(document.locator("strong").first()).toHaveText("Next responsible:");

    await expect(descriptionTab(testPage)).toHaveClass(/dv-active-tab/);
    await expect(sessionTabs(testPage)).toHaveCount(0);
    await expect(testPage.getByTestId("chat-input-area")).toHaveCount(0);

    const notice = testPage.getByTestId("sessionless-unassigned-notice");
    await expect(notice).toBeVisible();
    await expectNoticeBesideText(testPage);
    await expect(testPage.getByTestId("ensure-session-error-banner")).toHaveCount(0);
    await expect(testPage.getByTestId("ensure-session-error-retry")).toHaveCount(0);

    await testPage.getByTestId("sessionless-notice-details").click();
    await expect(testPage.getByTestId("sessionless-workspace-settings")).toHaveAttribute(
      "href",
      `/settings/workspaces/${seedData.workspaceId}`,
    );
    await testPage.keyboard.press("Escape");
    expect((await apiClient.listTaskSessions(task.id)).sessions).toHaveLength(0);

    await testPage.getByTestId("sessionless-start-agent").click();
    const session = new SessionPage(testPage);
    await expect(session.newSessionDialog()).toBeVisible();
    await session.newSessionPromptInput().fill("/e2e:simple-message");
    await session.newSessionStartButton().click();
    await expect(session.newSessionDialog()).toHaveCount(0);

    await expect(sessionTabs(testPage)).toHaveCount(1);
    await expect(sessionTabs(testPage)).toHaveClass(/dv-active-tab/);
    await expect(descriptionTab(testPage)).toHaveCount(1);
    await expect(testPage.getByTestId("sessionless-unassigned-notice")).toHaveCount(0);
    await expect
      .poll(async () => (await apiClient.listTaskSessions(task.id)).sessions.length)
      .toBe(1);
    expect(new URL(testPage.url()).pathname).toBe(`/t/${task.id}`);

    await descriptionTab(testPage).click();
    await expect(testPage.getByTestId("sessionless-task-description").locator("h2")).toHaveText(
      "Waiting on the owner",
    );
  });

  // @covers AC-TASKS-TASK-DESCRIPTION-VIEW-001.4
  test("scrolls a long description to its last line", async ({ testPage, apiClient, seedData }) => {
    test.setTimeout(120_000);
    const task = await apiClient.createTask(seedData.workspaceId, "Long unassigned task", {
      description: LONG_DESCRIPTION,
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      repository_ids: [seedData.repositoryId],
    });

    await testPage.goto(`/t/${task.id}`);

    const view = testPage.getByTestId("sessionless-task-view");
    const lastParagraph = view.getByText("Paragraph 60 of the plan body.");
    await expect(testPage.getByTestId("sessionless-start-agent")).toBeInViewport();
    await expect(lastParagraph).not.toBeInViewport();

    await view.hover();
    await expect
      .poll(async () => {
        await testPage.mouse.wheel(0, 600);
        return lastParagraph.evaluate((el) => {
          const rect = el.getBoundingClientRect();
          return rect.bottom > 0 && rect.bottom <= window.innerHeight;
        });
      })
      .toBe(true);
    const overflow = await testPage.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });

  // @covers AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.4
  test("ensures a session on the next open once a workspace default is set", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    test.setTimeout(120_000);
    const task = await apiClient.createTask(seedData.workspaceId, "Later assigned task", {
      description: DESCRIPTION,
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      repository_ids: [seedData.repositoryId],
    });

    await testPage.goto(`/t/${task.id}`);
    await expect(testPage.getByTestId("sessionless-unassigned-notice")).toBeVisible();

    await apiClient.updateWorkspace(seedData.workspaceId, {
      default_agent_profile_id: seedData.agentProfileId,
    });
    await testPage.reload();

    await expect
      .poll(async () => (await apiClient.listTaskSessions(task.id)).sessions.length)
      .toBe(1);
    await expect(testPage.getByTestId("sessionless-unassigned-notice")).toHaveCount(0);
  });
});
