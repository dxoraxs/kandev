"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconInfoCircle } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@kandev/ui/drawer";
import Link from "@/components/routing/app-link";
import { NewSessionDialog } from "./new-session-dialog";

/**
 * Phone form of the unassigned notice: a compact corner control that opens a
 * bottom drawer with the explanation and full-width touch actions.
 */
export function UnassignedNoticeDrawer({
  taskId,
  workspaceId,
  settingsWorkspaceId,
}: {
  taskId: string;
  workspaceId: string | null;
  settingsWorkspaceId: string | null;
}) {
  const { t } = useTranslation();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  return (
    <>
      <Drawer open={drawerOpen} onOpenChange={setDrawerOpen}>
        <DrawerTrigger asChild>
          <Button
            variant="outline"
            className="min-h-11 min-w-11 cursor-pointer gap-1.5 px-3 text-xs"
            data-testid="sessionless-unassigned-notice"
          >
            <IconInfoCircle className="h-4 w-4 shrink-0" aria-hidden />
            <span>{t("task:startAgent")}</span>
          </Button>
        </DrawerTrigger>
        <DrawerContent data-testid="sessionless-notice-drawer">
          <DrawerHeader className="text-left">
            <DrawerTitle>{t("task:noAgentProfileConfigured")}</DrawerTitle>
            <DrawerDescription>{t("task:noAgentProfileConfiguredDetail")}</DrawerDescription>
          </DrawerHeader>
          <div
            className="flex flex-col gap-2 px-4"
            style={{ paddingBottom: "calc(1rem + env(safe-area-inset-bottom, 0px))" }}
          >
            <Button
              className="min-h-11 w-full cursor-pointer"
              onClick={() => {
                setDrawerOpen(false);
                setDialogOpen(true);
              }}
              data-testid="sessionless-start-agent"
            >
              {t("task:startAgent")}
            </Button>
            {settingsWorkspaceId ? (
              <Button variant="outline" className="min-h-11 w-full cursor-pointer" asChild>
                <Link
                  href={`/settings/workspaces/${settingsWorkspaceId}`}
                  data-testid="sessionless-workspace-settings"
                >
                  {t("task:openWorkspaceSettings")}
                </Link>
              </Button>
            ) : null}
          </div>
        </DrawerContent>
      </Drawer>
      <NewSessionDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        taskId={taskId}
        workspaceId={workspaceId}
      />
    </>
  );
}
