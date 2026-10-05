import { describe, expect, it } from "vitest";
import { resolveSpaRoute } from "./spa-routes";

const WAITING_PATH = "/plans/waiting";
const WAITING_KIND = "plans-waiting";

describe("resolveSpaRoute plans waiting route", () => {
  it("resolves the waiting for owner page while planFiles is on", () => {
    expect(
      resolveSpaRoute(WAITING_PATH, new URLSearchParams(), { plansWaitingEnabled: true }),
    ).toEqual({ kind: WAITING_KIND });
  });

  it("tolerates a trailing slash", () => {
    expect(
      resolveSpaRoute(`${WAITING_PATH}/`, new URLSearchParams(), { plansWaitingEnabled: true }),
    ).toEqual({ kind: WAITING_KIND });
  });

  it("does not resolve the page while the flag is off", () => {
    expect(resolveSpaRoute(WAITING_PATH, new URLSearchParams()).kind).not.toBe(WAITING_KIND);
    expect(
      resolveSpaRoute(WAITING_PATH, new URLSearchParams(), { plansWaitingEnabled: false }).kind,
    ).toBe("kanban");
  });

  it("does not claim other paths under /plans", () => {
    expect(
      resolveSpaRoute("/plans/other", new URLSearchParams(), { plansWaitingEnabled: true }).kind,
    ).not.toBe(WAITING_KIND);
  });
});
