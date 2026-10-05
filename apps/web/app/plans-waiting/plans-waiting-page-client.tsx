"use client";

import { useTranslation } from "react-i18next";
import { Button } from "@kandev/ui/button";
import { PageShell } from "@/components/page-shell";
import { WaitingOwnerList } from "@/components/plans-waiting/waiting-owner-list";
import { WaitingOwnerTable } from "@/components/plans-waiting/waiting-owner-table";
import { useWaitingOwner, type WaitingOwnerState } from "@/hooks/domains/plans/use-waiting-owner";
import { useResponsiveBreakpoint } from "@/hooks/use-responsive-breakpoint";
import { cn } from "@/lib/utils";

function FailedWorkspaces({ names }: { names: string[] }) {
  const { t } = useTranslation();
  if (names.length === 0) return null;
  return (
    <p
      role="alert"
      className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm"
      data-testid="plans-waiting-failed"
    >
      {t("planFiles:waitingFailed", { names: names.join(", ") })}
    </p>
  );
}

function LoadError({ onRetry, touch }: { onRetry: () => void; touch: boolean }) {
  const { t } = useTranslation();
  return (
    <div
      role="alert"
      className="flex flex-wrap items-center gap-3 text-sm"
      data-testid="plans-waiting-error"
    >
      <span>{t("planFiles:waitingLoadError")}</span>
      <Button
        type="button"
        variant="outline"
        className={cn("cursor-pointer", touch && "min-h-11")}
        onClick={onRetry}
      >
        {t("planFiles:waitingRetry")}
      </Button>
    </div>
  );
}

function WaitingOwnerBody({ waiting, touch }: { waiting: WaitingOwnerState; touch: boolean }) {
  const { t } = useTranslation();
  const { status, items } = waiting;
  if (status === "loading" && items.length === 0) {
    return (
      <p role="status" aria-live="polite" className="text-sm text-muted-foreground">
        {t("common:loading")}
      </p>
    );
  }
  return (
    <>
      {status === "error" && <LoadError onRetry={waiting.reload} touch={touch} />}
      {items.length > 0 &&
        (touch ? <WaitingOwnerList items={items} /> : <WaitingOwnerTable items={items} />)}
      {items.length === 0 && status === "ready" && (
        <p className="text-sm text-muted-foreground" data-testid="plans-waiting-empty">
          {t("planFiles:waitingEmpty")}
        </p>
      )}
    </>
  );
}

/** Plans that wait for the owner across every accessible workspace. */
export function PlansWaitingPageClient() {
  const { t } = useTranslation();
  const { isMobile } = useResponsiveBreakpoint();
  const waiting = useWaitingOwner();
  const title =
    waiting.status === "ready" || waiting.items.length > 0
      ? t("planFiles:waitingTitleCount", { total: waiting.items.length })
      : t("planFiles:waitingTitle");
  return (
    <PageShell title={title} contentClassName="space-y-4 p-4 md:p-6">
      <div className="space-y-4" data-testid="plans-waiting-page">
        <FailedWorkspaces names={waiting.failedWorkspaces.map((w) => w.workspace_name)} />
        <WaitingOwnerBody waiting={waiting} touch={isMobile} />
      </div>
    </PageShell>
  );
}
