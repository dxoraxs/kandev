"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconInfoCircle } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { Drawer, DrawerContent, DrawerHeader, DrawerTitle, DrawerTrigger } from "@kandev/ui/drawer";
import { Textarea } from "@kandev/ui/textarea";
import type { PlanDecisionState } from "@/hooks/domains/plans/use-plan-decision";
import {
  planDecisionBody,
  planDecisionReady,
  type PlanDecisionIntent,
} from "./plan-decision-intent";

const TOUCH_BUTTON = "min-h-11 w-full cursor-pointer";

const ACTIONS: { intent: PlanDecisionIntent; labelKey: string; testId: string }[] = [
  { intent: "accept", labelKey: "planFiles:decisionAccept", testId: "plan-decision-drawer-accept" },
  {
    intent: "acceptQueue",
    labelKey: "planFiles:decisionAcceptQueue",
    testId: "plan-decision-drawer-accept-queue",
  },
  { intent: "return", labelKey: "planFiles:decisionReturn", testId: "plan-decision-drawer-return" },
];

/**
 * Phone form of the decision bar: a full-width trigger that opens a bottom
 * drawer with the comment field and one full-width touch button per decision.
 */
export function PlanDecisionDrawer({ decision }: { decision: PlanDecisionState }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [comment, setComment] = useState("");
  const choose = async (intent: PlanDecisionIntent) => {
    if (await decision.submit(planDecisionBody(intent, comment))) setOpen(false);
  };
  return (
    <Drawer open={open} onOpenChange={setOpen}>
      <DrawerTrigger asChild>
        <Button
          variant="outline"
          className="min-h-11 w-full cursor-pointer justify-start gap-2 px-3 text-xs"
          data-testid="plan-decision-trigger"
        >
          <IconInfoCircle className="h-4 w-4 shrink-0" aria-hidden />
          <span className="truncate">{t("planFiles:decisionWaiting")}</span>
        </Button>
      </DrawerTrigger>
      <DrawerContent data-testid="plan-decision-drawer">
        <DrawerHeader className="text-left">
          <DrawerTitle>{t("planFiles:decisionTitle")}</DrawerTitle>
        </DrawerHeader>
        <div
          className="flex flex-col gap-2 px-4"
          style={{ paddingBottom: "calc(1rem + env(safe-area-inset-bottom, 0px))" }}
        >
          <Textarea
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            aria-label={t("planFiles:decisionComment")}
            placeholder={t("planFiles:decisionCommentPlaceholder")}
            maxLength={500}
            data-testid="plan-decision-comment"
          />
          {decision.error ? (
            <p role="alert" className="text-xs text-destructive" data-testid="plan-decision-error">
              {decision.error}
            </p>
          ) : null}
          {ACTIONS.map(({ intent, labelKey, testId }) => (
            <Button
              key={intent}
              variant={intent === "accept" ? "default" : "outline"}
              className={TOUCH_BUTTON}
              disabled={decision.pending || !planDecisionReady(intent, comment)}
              onClick={() => choose(intent)}
              data-testid={testId}
            >
              {t(labelKey)}
            </Button>
          ))}
        </div>
      </DrawerContent>
    </Drawer>
  );
}
