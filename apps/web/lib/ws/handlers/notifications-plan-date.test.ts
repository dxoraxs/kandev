import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { StoreApi } from "zustand";
import type { AppState } from "@/lib/state/store";
import { NOTIFICATION_EVENT_PLAN_FILE_DATE_REACHED } from "@/lib/notifications/events";
import type { BackendMessageMap } from "@/lib/types/backend";
import { registerNotificationsHandlers } from "./notifications";

vi.mock("@/lib/notifications/sound", () => ({
  playWaitingForInputSound: vi.fn(),
}));

const notificationMock = vi.fn();

function makeStore() {
  const state = { tasks: { activeTaskId: "other" } } as unknown as AppState;
  return { getState: () => state } as unknown as StoreApi<AppState>;
}

function handler() {
  const handlers = registerNotificationsHandlers(makeStore());
  return handlers[NOTIFICATION_EVENT_PLAN_FILE_DATE_REACHED]!;
}

describe("plan date notification delivery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      "Notification",
      Object.assign(notificationMock, { permission: "granted" as NotificationPermission }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("uses the event name the backend sends", () => {
    expect(NOTIFICATION_EVENT_PLAN_FILE_DATE_REACHED).toBe("plan_file.date_reached");
  });

  it("shows the title and body the backend sends", () => {
    handler()({
      type: "notification",
      action: "plan_file.date_reached",
      payload: { title: "Plan date reached", body: '"Launch plan" is waiting for you.' },
    } as BackendMessageMap["plan_file.date_reached"]);

    expect(notificationMock).toHaveBeenCalledWith("Plan date reached", {
      body: '"Launch plan" is waiting for you.',
    });
  });

  it("falls back to catalog copy when the payload carries none", () => {
    handler()({
      type: "notification",
      action: "plan_file.date_reached",
      payload: { title: "", body: "" },
    } as BackendMessageMap["plan_file.date_reached"]);

    expect(notificationMock).toHaveBeenCalledWith("Plan date reached", {
      body: "A plan is waiting for you.",
    });
  });
});
