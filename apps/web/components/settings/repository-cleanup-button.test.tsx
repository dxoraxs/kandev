import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { StateProvider } from "@/components/state-provider";
import { defaultState } from "@/lib/state/default-state";

const startRepositoryMaintenanceTask = vi.fn();
vi.mock("@/lib/api/domains/repository-maintenance-api", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/api/domains/repository-maintenance-api")>()),
  startRepositoryMaintenanceTask: (...a: unknown[]) => startRepositoryMaintenanceTask(...a),
}));
let mobile = false;
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: mobile, isFinePointer: !mobile }),
}));
vi.mock("@/components/toast-provider", () => ({ useToast: () => ({ toast: vi.fn() }) }));
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ push: vi.fn() }) }));

import { RepositoryCleanupButton } from "./repository-cleanup-button";

const BUTTON = "repository-cleanup-button";

function renderButton(
  props: {
    sourceType?: string;
    readOnly?: boolean;
    flag?: boolean;
    onParentClick?: () => void;
  } = {},
) {
  const { sourceType = "local", readOnly = false, flag = true, onParentClick } = props;
  return render(
    <StateProvider
      initialState={{ features: { ...defaultState.features, repositoryCleanup: flag } }}
    >
      <div onClick={onParentClick}>
        <RepositoryCleanupButton
          repository={{ id: "r1", name: "city_companion", source_type: sourceType }}
          readOnly={readOnly}
        />
      </div>
    </StateProvider>,
  );
}

beforeEach(() => {
  mobile = false;
  startRepositoryMaintenanceTask.mockReset();
});
afterEach(cleanup);

describe("RepositoryCleanupButton visibility", () => {
  it("renders an icon-only button with an accessible name for a local repository", () => {
    renderButton();
    const button = screen.getByTestId(BUTTON);
    expect(button.getAttribute("aria-label")).toBe("Clean up repository");
    expect(button.textContent).toBe("");
    expect(button.className).toContain("size-7");
    expect(button.className).toContain("max-md:size-11");
  });

  it("is absent when the runtime flag is off", () => {
    renderButton({ flag: false });
    expect(screen.queryByTestId(BUTTON)).toBeNull();
  });

  it("is absent for a remote repository", () => {
    renderButton({ sourceType: "remote" });
    expect(screen.queryByTestId(BUTTON)).toBeNull();
  });

  it("is absent in read-only mode", () => {
    renderButton({ readOnly: true });
    expect(screen.queryByTestId(BUTTON)).toBeNull();
  });
});

describe("RepositoryCleanupButton interaction", () => {
  it("opens the confirmation without letting the click reach the card", () => {
    const onParentClick = vi.fn();
    renderButton({ onParentClick });
    fireEvent.click(screen.getByTestId(BUTTON));
    expect(onParentClick).not.toHaveBeenCalled();
    expect(screen.getByTestId("repository-cleanup-dialog").textContent).toContain(
      "Clean up city_companion?",
    );
    expect(startRepositoryMaintenanceTask).not.toHaveBeenCalled();
  });

  it("clicks inside the confirmation do not reach the card", () => {
    const onParentClick = vi.fn();
    renderButton({ onParentClick });
    fireEvent.click(screen.getByTestId(BUTTON));
    fireEvent.click(screen.getByTestId("repository-cleanup-dialog"));
    fireEvent.click(screen.getByTestId("repository-cleanup-cancel"));
    expect(onParentClick).not.toHaveBeenCalled();
  });

  it("closing the confirmation with Cancel creates nothing", () => {
    renderButton();
    fireEvent.click(screen.getByTestId(BUTTON));
    fireEvent.click(screen.getByTestId("repository-cleanup-cancel"));
    expect(screen.queryByTestId("repository-cleanup-dialog")).toBeNull();
    expect(startRepositoryMaintenanceTask).not.toHaveBeenCalled();
  });
});
