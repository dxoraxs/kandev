import { describe, expect, it } from "vitest";
import { cardDisplayFromMetadata, dateTagTone } from "./card-display";

const DATE = "2026-10-05";
const wrap = (card_display: unknown) => ({ card_display });

describe("cardDisplayFromMetadata", () => {
  it("returns undefined for absent or non-object input", () => {
    for (const value of [undefined, null, "x", 3, true, [], {}, wrap(null), wrap("x"), wrap([])]) {
      expect(cardDisplayFromMetadata(value)).toBeUndefined();
    }
  });

  it("parses every valid field", () => {
    expect(
      cardDisplayFromMetadata(
        wrap({
          date: DATE,
          date_kind: "waiting",
          executor: { name: "Claude", kind: "person" },
          progress: { done: 1, total: 3 },
        }),
      ),
    ).toEqual({
      date: { iso: DATE, kind: "waiting" },
      executor: { name: "Claude", kind: "person" },
      progress: { done: 1, total: 3 },
    });
  });

  it("defaults date_kind to due and executor kind to agent", () => {
    expect(cardDisplayFromMetadata(wrap({ date: DATE, date_kind: "later" }))).toEqual({
      date: { iso: DATE, kind: "due" },
    });
    expect(cardDisplayFromMetadata(wrap({ executor: { name: "Bo", kind: "robot" } }))).toEqual({
      executor: { name: "Bo", kind: "agent" },
    });
  });

  it("rejects malformed and impossible dates", () => {
    for (const date of ["2026-02-30", "2026-13-01", "2026-1-1", "10/05/2026", 20261005, null, ""]) {
      expect(cardDisplayFromMetadata(wrap({ date }))).toBeUndefined();
    }
  });

  it("accepts a leap day only in a leap year", () => {
    expect(cardDisplayFromMetadata(wrap({ date: "2028-02-29" }))?.date?.iso).toBe("2028-02-29");
    expect(cardDisplayFromMetadata(wrap({ date: "2027-02-29" }))).toBeUndefined();
  });

  it("trims the executor name and enforces 1 to 40 characters", () => {
    expect(cardDisplayFromMetadata(wrap({ executor: { name: "  Ann  " } }))?.executor?.name).toBe(
      "Ann",
    );
    expect(cardDisplayFromMetadata(wrap({ executor: { name: "a".repeat(40) } }))).toBeDefined();
    for (const name of ["   ", "", "a".repeat(41), 5, null]) {
      expect(cardDisplayFromMetadata(wrap({ executor: { name } }))).toBeUndefined();
    }
    expect(cardDisplayFromMetadata(wrap({ executor: "Ann" }))).toBeUndefined();
  });

  it("validates progress as integers with 0 <= done <= total and total > 0", () => {
    expect(cardDisplayFromMetadata(wrap({ progress: { done: 0, total: 1 } }))?.progress).toEqual({
      done: 0,
      total: 1,
    });
    for (const progress of [
      { done: 4, total: 3 },
      { done: -1, total: 3 },
      { done: 0, total: 0 },
      { done: 1.5, total: 3 },
      { done: "1", total: 3 },
      { done: 1 },
      [],
      null,
    ]) {
      expect(cardDisplayFromMetadata(wrap({ progress }))).toBeUndefined();
    }
  });

  it("keeps valid fields when a sibling is invalid", () => {
    expect(
      cardDisplayFromMetadata(wrap({ date: "nope", progress: { done: 1, total: 2 } })),
    ).toEqual({ progress: { done: 1, total: 2 } });
  });
});

describe("dateTagTone", () => {
  const today = new Date(2026, 9, 5, 15, 30);

  it("marks past days danger", () => {
    expect(dateTagTone("2026-10-04", "due", today)).toBe("danger");
    expect(dateTagTone("2025-12-31", "waiting", today)).toBe("danger");
  });

  it("marks today and tomorrow warning", () => {
    expect(dateTagTone(DATE, "due", today)).toBe("warning");
    expect(dateTagTone("2026-10-06", "due", today)).toBe("warning");
  });

  it("marks later days neutral", () => {
    expect(dateTagTone("2026-10-07", "due", today)).toBe("neutral");
  });

  it("compares calendar days across month and year boundaries", () => {
    expect(dateTagTone("2027-01-01", "due", new Date(2026, 11, 31, 23, 59))).toBe("warning");
    expect(dateTagTone("2026-11-01", "due", new Date(2026, 9, 31, 0, 1))).toBe("warning");
  });

  it("never shows danger for deferred", () => {
    expect(dateTagTone("2026-10-01", "deferred", today)).toBe("warning");
    expect(dateTagTone(DATE, "deferred", today)).toBe("warning");
    expect(dateTagTone("2026-12-01", "deferred", today)).toBe("neutral");
  });
});
