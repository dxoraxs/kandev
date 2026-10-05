import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";
import type { Repository } from "@/lib/types/http";
import type { TaskListingPage } from "@/lib/task-listing/view-navigation";

const toast = vi.fn();
vi.mock("@/components/toast-provider", () => ({ useToast: () => ({ toast }) }));
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: false, isFinePointer: true }),
}));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push: vi.fn() }) }));

import {
  BoardCleanupButton,
  MobileCleanupEntry,
  useBoardCleanupAction,
} from "./board-cleanup-action";

function repo(id: string, sourceType = "local"): Repository {
  return { id, name: id, source_type: sourceType } as Repository;
}

function Harness({
  page = "kanban",
  repositories,
  selected = null,
}: {
  page?: TaskListingPage;
  repositories: Repository[];
  selected?: string | null;
}) {
  const action = useBoardCleanupAction({
    currentPage: page,
    repositories,
    selectedRepositoryId: selected,
  });
  if (!action) return <span data-testid="no-action" />;
  return (
    <>
      <BoardCleanupButton action={action} />
      {action.dialog}
    </>
  );
}

function renderHarness(props: Parameters<typeof Harness>[0], flag = true) {
  return render(
    <StateProvider
      initialState={{ features: { ...defaultState.features, repositoryCleanup: flag } }}
    >
      <Harness {...props} />
    </StateProvider>,
  );
}

const BUTTON = "board-cleanup-button";

beforeEach(() => toast.mockReset());
afterEach(cleanup);

describe("board cleanup action", () => {
  it("is absent while the runtime flag is off", () => {
    renderHarness({ repositories: [repo("a")] }, false);
    expect(screen.getByTestId("no-action")).toBeTruthy();
  });

  it("is absent outside the kanban page", () => {
    renderHarness({ page: "tasks", repositories: [repo("a")] });
    expect(screen.getByTestId("no-action")).toBeTruthy();
  });

  it("is absent without a local repository", () => {
    renderHarness({ repositories: [repo("r", "github")] });
    expect(screen.getByTestId("no-action")).toBeTruthy();
  });

  it("opens the confirmation for the selected repository", () => {
    renderHarness({ repositories: [repo("alpha"), repo("beta")], selected: "beta" });
    const button = screen.getByTestId(BUTTON);
    expect(button.getAttribute("aria-label")).toBe("Clean up repository");
    expect(button.getAttribute("aria-disabled")).toBeNull();
    fireEvent.click(button);
    expect(screen.getByTestId("repository-cleanup-dialog").textContent).toContain("Clean up beta?");
  });

  it("asks for a repository when several are local and none is selected", () => {
    renderHarness({ repositories: [repo("alpha"), repo("beta")] });
    const button = screen.getByTestId(BUTTON);
    expect(button.getAttribute("aria-disabled")).toBe("true");
    fireEvent.click(button);
    expect(screen.queryByTestId("repository-cleanup-dialog")).toBeNull();
    expect(toast).toHaveBeenCalledWith(
      expect.objectContaining({ description: "Choose a repository in the filter to clean it up." }),
    );
  });
});

describe("MobileCleanupEntry", () => {
  it("is a labelled entry that opens the confirmation", () => {
    const onOpen = vi.fn();
    render(<MobileCleanupEntry onOpen={onOpen} />);
    const entry = screen.getByTestId("mobile-display-cleanup");
    expect(entry.textContent).toContain("Clean up repository");
    expect(entry.className).toContain("min-h-11");
    fireEvent.click(entry);
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("is disabled with the hint until a repository is chosen", () => {
    render(<MobileCleanupEntry onOpen={undefined} />);
    const entry = screen.getByTestId("mobile-display-cleanup") as HTMLButtonElement;
    expect(entry.disabled).toBe(true);
    expect(screen.getByText("Choose a repository in the filter to clean it up.")).toBeTruthy();
  });
});
