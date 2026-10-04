import { mkdirSync } from "node:fs";
import path from "node:path";
import type { Locator, Page } from "@playwright/test";

/** Formats a calendar day in the browser's local time zone as YYYY-MM-DD. */
export async function localIsoDay(page: Page, offsetDays: number): Promise<string> {
  return page.evaluate((offset) => {
    const d = new Date();
    d.setDate(d.getDate() + offset);
    const p = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
  }, offsetDays);
}

export async function backgroundColor(locator: Locator): Promise<string> {
  return locator.evaluate((node) => getComputedStyle(node).backgroundColor);
}

/** Writes a card screenshot only when KANDEV_E2E_SHOTS_DIR is set. */
export async function maybeShoot(card: Locator, name: string): Promise<void> {
  const dir = process.env.KANDEV_E2E_SHOTS_DIR;
  if (!dir) return;
  mkdirSync(dir, { recursive: true });
  await card.screenshot({ path: path.join(dir, name) });
}
