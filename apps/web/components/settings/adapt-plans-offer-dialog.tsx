"use client";

import { useTranslation } from "react-i18next";
import { IconLoader2 } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@kandev/ui/dialog";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@kandev/ui/drawer";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import { useMaintenanceTaskStart } from "@/hooks/domains/settings/use-maintenance-task-start";
import { settingsActionClassName } from "./settings-control";

export type PlanAdaptationOffer = {
  repositoryId: string;
  repositoryName: string;
  count: number;
  /** True when plan sync is not configured, so adapting also creates the board. */
  boardWillBeCreated: boolean;
};

type OfferDialogProps = {
  offer: PlanAdaptationOffer | null;
  onDismiss: () => void;
};

type ActionsProps = {
  pending: boolean;
  stacked: boolean;
  onNotNow: () => void;
  onAdapt: () => void;
};

function OfferActions({ pending, stacked, onNotNow, onAdapt }: ActionsProps) {
  const { t } = useTranslation();
  const width = stacked ? "w-full" : "w-auto";
  const notNow = (
    <Button
      key="not-now"
      type="button"
      variant="outline"
      disabled={pending}
      onClick={onNotNow}
      className={settingsActionClassName(`${width} cursor-pointer`)}
      data-testid="adapt-plans-not-now"
    >
      {t("planFiles:notNow")}
    </Button>
  );
  const adapt = (
    <Button
      key="adapt"
      type="button"
      disabled={pending}
      onClick={onAdapt}
      className={settingsActionClassName(`${width} cursor-pointer`)}
      data-testid="adapt-plans-confirm"
    >
      {pending && <IconLoader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />}
      {t("planFiles:adaptWithAgent")}
    </Button>
  );
  // The primary action comes first in the phone column and last in the desktop row.
  return stacked ? [adapt, notNow] : [notNow, adapt];
}

function OfferExtras({ offer, pending }: { offer: PlanAdaptationOffer; pending: boolean }) {
  const { t } = useTranslation();
  return (
    <>
      {offer.boardWillBeCreated && (
        <p className="text-xs text-muted-foreground" data-testid="adapt-plans-offer-board">
          {t("planFiles:boardWillBeCreated")}
        </p>
      )}
      <div role="status" className="sr-only">
        {pending ? t("planFiles:adaptStarting") : ""}
      </div>
    </>
  );
}

/**
 * Offered once after a repository with unadapted plan files is added: a
 * centered dialog on wider screens, an inset bottom drawer on phones.
 */
export function AdaptPlansOfferDialog({ offer, onDismiss }: OfferDialogProps) {
  const { t } = useTranslation();
  const { isMobile } = useResponsiveBreakpoint();
  const { start, pendingRepositoryId } = useMaintenanceTaskStart();
  const pending = pendingRepositoryId !== null;

  const adapt = async () => {
    if (!offer) return;
    if (await start(offer.repositoryId, "plan_adaptation")) onDismiss();
  };
  const onOpenChange = (open: boolean) => {
    if (!open && !pending) onDismiss();
  };
  const actions = (
    <OfferActions
      pending={pending}
      stacked={isMobile}
      onNotNow={onDismiss}
      onAdapt={() => void adapt()}
    />
  );
  const title = offer ? t("planFiles:offerTitle", { name: offer.repositoryName }) : "";
  const body = offer ? t("planFiles:offerBody", { count: offer.count }) : "";

  if (isMobile) {
    return (
      <Drawer open={offer !== null} onOpenChange={onOpenChange}>
        <DrawerContent data-testid="adapt-plans-offer" className="max-h-[85dvh]">
          {offer && (
            <>
              <DrawerHeader className="text-left">
                <DrawerTitle>{title}</DrawerTitle>
                <DrawerDescription>{body}</DrawerDescription>
              </DrawerHeader>
              <div className="px-4">
                <OfferExtras offer={offer} pending={pending} />
              </div>
              <DrawerFooter className="pb-[calc(1rem+env(safe-area-inset-bottom))]">
                {actions}
              </DrawerFooter>
            </>
          )}
        </DrawerContent>
      </Drawer>
    );
  }
  return (
    <Dialog open={offer !== null} onOpenChange={onOpenChange}>
      <DialogContent data-testid="adapt-plans-offer">
        {offer && (
          <>
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
              <DialogDescription>{body}</DialogDescription>
            </DialogHeader>
            <OfferExtras offer={offer} pending={pending} />
            <DialogFooter>{actions}</DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
