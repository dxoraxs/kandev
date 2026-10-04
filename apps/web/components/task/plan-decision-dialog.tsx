"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@kandev/ui/dialog";
import { Textarea } from "@kandev/ui/textarea";
import type { PlanDecisionState } from "@/hooks/domains/plans/use-plan-decision";
import {
  planDecisionBody,
  planDecisionReady,
  type PlanDecisionIntent,
} from "./plan-decision-intent";

const TITLE_KEY = {
  return: "planFiles:decisionDialogReturn",
  accept: "planFiles:decisionDialogAccept",
  acceptQueue: "planFiles:decisionDialogAcceptQueue",
} as const;

const CONFIRM_KEY = {
  return: "planFiles:decisionReturn",
  accept: "planFiles:decisionAccept",
  acceptQueue: "planFiles:decisionAcceptQueue",
} as const;

/** Desktop confirmation of one decision, with the comment the owner note records. */
export function PlanDecisionDialog({
  intent,
  onClose,
  decision,
}: {
  /** The open decision, or null while the dialog is closed. */
  intent: PlanDecisionIntent | null;
  onClose: () => void;
  decision: PlanDecisionState;
}) {
  return (
    <Dialog open={intent !== null} onOpenChange={(open) => !open && onClose()}>
      {intent ? (
        <PlanDecisionDialogBody intent={intent} onClose={onClose} decision={decision} />
      ) : null}
    </Dialog>
  );
}

function PlanDecisionDialogBody({
  intent,
  onClose,
  decision,
}: {
  intent: PlanDecisionIntent;
  onClose: () => void;
  decision: PlanDecisionState;
}) {
  const { t } = useTranslation();
  const [comment, setComment] = useState("");
  const confirm = async () => {
    if (await decision.submit(planDecisionBody(intent, comment))) onClose();
  };
  return (
    <DialogContent enterConfirms={false} data-testid="plan-decision-dialog">
      <DialogHeader>
        <DialogTitle>{t(TITLE_KEY[intent])}</DialogTitle>
      </DialogHeader>
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
      <DialogFooter>
        <Button variant="outline" className="cursor-pointer" onClick={onClose}>
          {t("planFiles:decisionCancel")}
        </Button>
        <Button
          className="cursor-pointer"
          disabled={decision.pending || !planDecisionReady(intent, comment)}
          onClick={confirm}
          data-testid="plan-decision-confirm"
        >
          {t(CONFIRM_KEY[intent])}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}
