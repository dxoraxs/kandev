import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

const replace = vi.fn();
vi.mock("@/lib/routing/client-router", () => ({ useRouter: () => ({ replace }) }));
vi.mock("@/app/plans-waiting/plans-waiting-page-client", () => ({
  PlansWaitingPageClient: () => <div data-testid="plans-waiting-page-stub" />,
}));

import { PlansWaitingRoute } from "./plans-waiting-route";

describe("PlansWaitingRoute", () => {
  afterEach(() => {
    cleanup();
    replace.mockReset();
  });

  it("renders the page while the feature is on", async () => {
    render(<PlansWaitingRoute enabled />);
    expect(await screen.findByTestId("plans-waiting-page-stub")).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("sends the visitor home while the feature is off", () => {
    render(<PlansWaitingRoute enabled={false} />);
    expect(screen.queryByTestId("plans-waiting-page-stub")).toBeNull();
    expect(replace).toHaveBeenCalledWith("/");
  });
});
