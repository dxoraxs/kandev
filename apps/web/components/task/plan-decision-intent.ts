import type { PlanDecisionBody } from "@/lib/api/domains/plan-files-api";

/** The three ways an owner can answer a plan waiting for them. */
export type PlanDecisionIntent = "return" | "accept" | "acceptQueue";

/** The request body of an intent; an empty comment is left out. */
export function planDecisionBody(intent: PlanDecisionIntent, comment: string): PlanDecisionBody {
  const text = comment.trim();
  const withComment = text ? { comment: text } : {};
  if (intent === "return") return { action: "return", ...withComment };
  return { action: "accept", result: intent === "accept" ? "done" : "queued", ...withComment };
}

/** Return needs a comment; an accept does not. */
export function planDecisionReady(intent: PlanDecisionIntent, comment: string): boolean {
  return intent !== "return" || comment.trim() !== "";
}
