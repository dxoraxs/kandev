"use client";

import { IconUserCheck } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@kandev/ui/badge";
import { Button } from "@kandev/ui/button";
import { AppSidebarNavItem } from "@/components/app-sidebar/app-sidebar-nav-item";
import Link from "@/components/routing/app-link";
import { useFeature } from "@/hooks/domains/features/use-feature";
import { useWaitingOwner } from "@/hooks/domains/plans/use-waiting-owner";
import { PLANS_WAITING_HREF } from "@/lib/navigation/plans-waiting-destination";

function SidebarItem({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation();
  const { items } = useWaitingOwner();
  return (
    <AppSidebarNavItem
      icon={IconUserCheck}
      label={t("planFiles:waitingTitle")}
      href={PLANS_WAITING_HREF}
      badge={items.length}
      collapsed={collapsed}
      testId="sidebar-plans-waiting"
    />
  );
}

/** Primary-navigation entry for the Waiting for owner page; absent while planFiles is off. */
export function PlansWaitingSidebarItem({ collapsed }: { collapsed: boolean }) {
  if (!useFeature("planFiles")) return null;
  return <SidebarItem collapsed={collapsed} />;
}

function MobileRow({ onNavigate }: { onNavigate: () => void }) {
  const { t } = useTranslation();
  const { items } = useWaitingOwner();
  return (
    <Button
      asChild
      variant="outline"
      className="h-11 w-full cursor-pointer justify-start gap-3 px-3"
    >
      <Link
        href={PLANS_WAITING_HREF}
        onClick={onNavigate}
        data-testid="mobile-sidebar-plans-waiting"
      >
        <IconUserCheck className="h-4 w-4 shrink-0" />
        <span className="flex-1 text-left">{t("planFiles:waitingTitle")}</span>
        {items.length > 0 && <Badge>{items.length}</Badge>}
      </Link>
    </Button>
  );
}

/** Phone navigation-sheet row for the Waiting for owner page; absent while planFiles is off. */
export function PlansWaitingMobileRow({ onNavigate }: { onNavigate: () => void }) {
  if (!useFeature("planFiles")) return null;
  return <MobileRow onNavigate={onNavigate} />;
}
