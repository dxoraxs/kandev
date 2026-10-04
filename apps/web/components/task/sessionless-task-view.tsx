"use client";

import { createContext, useContext, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { IconHelpCircle, IconInfoCircle } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@kandev/ui/popover";
import Link from "@/components/routing/app-link";
import { GridSpinner } from "@/components/grid-spinner";
import { MemoizedMarkdown } from "@/components/shared/memoized-markdown";
import { useTask } from "@/hooks/use-task";
import type { EnsureTaskSessionStatus } from "@/hooks/domains/session/use-ensure-task-session";
import { NewSessionDialog } from "./new-session-dialog";
import { UnassignedNoticeDrawer } from "./sessionless-notice-drawer";
import { resolveSessionlessTaskView } from "./sessionless-task-view-model";

export type SessionlessTaskContextValue = {
  taskId: string;
  workspaceId: string | null;
  status: EnsureTaskSessionStatus;
};

/** Ensure status of the open task, for surfaces that render a task with no session. */
export const SessionlessTaskContext = createContext<SessionlessTaskContextValue | null>(null);

/** The open task's sessionless context, or null when another task owns the provider. */
export function useSessionlessTaskContext(
  taskId: string | null | undefined,
): SessionlessTaskContextValue | null {
  const context = useContext(SessionlessTaskContext);
  return context && context.taskId === taskId ? context : null;
}

/**
 * Returns the sessionless view props when a transcript host shows a task with no
 * session; null when a session exists, a launch error owns the panel, or the
 * provider belongs to another task.
 */
export function useSessionlessTaskView(input: {
  taskId: string | null | undefined;
  hasSession: boolean;
  launchErrorOwned: boolean;
}): SessionlessTaskContextValue | null {
  const context = useSessionlessTaskContext(input.taskId);
  if (input.hasSession || input.launchErrorOwned) return null;
  return context;
}

type SessionlessPresentation = "desktop" | "mobile";

/** The task description as a document, with the ensure status in its top-right corner. */
export function SessionlessTaskView({
  taskId,
  workspaceId,
  status,
  presentation = "desktop",
}: SessionlessTaskContextValue & { presentation?: SessionlessPresentation }) {
  const task = useTask(taskId);
  const view = resolveSessionlessTaskView({
    description: task?.description,
    status,
    workspaceId,
  });
  let corner: ReactNode = null;
  if (view.statusLine === "preparing") corner = <PreparingCorner />;
  if (view.statusLine === "unassigned") {
    const Notice = presentation === "mobile" ? UnassignedNoticeDrawer : UnassignedCornerNotice;
    corner = (
      <Notice
        taskId={taskId}
        workspaceId={workspaceId}
        settingsWorkspaceId={view.settingsWorkspaceId}
      />
    );
  }
  return (
    <TaskDescriptionDocument
      taskId={taskId}
      description={view.description}
      corner={corner}
      cornerPlacement={presentation === "mobile" ? "float" : "responsive"}
    />
  );
}

const CORNER_CLASS = {
  responsive:
    "mb-3 w-full @2xl/task-description:float-right @2xl/task-description:ml-4 @2xl/task-description:w-72",
  float: "float-right mb-2 ml-3",
} as const;

/**
 * The description rendered as a Markdown document. With `responsive` placement
 * the corner content floats to the top-right on wide containers so the text
 * wraps around it, and stacks above the document on narrow ones; `float`
 * always floats a compact control.
 */
export function TaskDescriptionDocument({
  taskId,
  description,
  corner,
  cornerPlacement = "responsive",
}: {
  taskId: string;
  description: string | null;
  corner?: ReactNode;
  cornerPlacement?: keyof typeof CORNER_CLASS;
}) {
  const { t } = useTranslation();
  return (
    <div
      className="@container/task-description h-full min-h-0 flex-1 overflow-y-auto px-4 py-4"
      data-testid="sessionless-task-view"
    >
      {corner ? (
        <div className={CORNER_CLASS[cornerPlacement]} data-testid="sessionless-corner">
          {corner}
        </div>
      ) : null}
      {description ? (
        <div className="markdown-body min-w-0" data-testid="sessionless-task-description">
          <MemoizedMarkdown content={description} taskId={taskId} />
        </div>
      ) : (
        <p className="text-sm text-muted-foreground" data-testid="sessionless-task-no-description">
          {t("task:sessionlessNoDescription")}
        </p>
      )}
    </div>
  );
}

function PreparingCorner() {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground" role="status">
      <GridSpinner className="text-primary" ariaLabel={t("task:preparingWorkspace2")} />
      <span>{t("task:preparingWorkspace2")}</span>
    </div>
  );
}

function UnassignedCornerNotice({
  taskId,
  workspaceId,
  settingsWorkspaceId,
}: {
  taskId: string;
  workspaceId: string | null;
  settingsWorkspaceId: string | null;
}) {
  const { t } = useTranslation();
  const [dialogOpen, setDialogOpen] = useState(false);
  return (
    <div
      className="rounded-md border bg-card p-3 text-xs text-card-foreground shadow-sm"
      data-testid="sessionless-unassigned-notice"
    >
      <div className="flex items-start gap-2">
        <IconInfoCircle className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
        <span className="flex-1 font-medium">{t("task:noAgentProfileConfigured")}</span>
        <UnassignedNoticeDetails settingsWorkspaceId={settingsWorkspaceId} />
      </div>
      <Button
        size="sm"
        className="mt-2 cursor-pointer"
        onClick={() => setDialogOpen(true)}
        data-testid="sessionless-start-agent"
      >
        {t("task:startAgent")}
      </Button>
      <NewSessionDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        taskId={taskId}
        workspaceId={workspaceId}
      />
    </div>
  );
}

function UnassignedNoticeDetails({ settingsWorkspaceId }: { settingsWorkspaceId: string | null }) {
  const { t } = useTranslation();
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6 shrink-0 cursor-pointer"
          aria-label={t("task:unassignedNoticeDetails")}
          data-testid="sessionless-notice-details"
        >
          <IconHelpCircle className="h-4 w-4" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 space-y-2 text-xs">
        <p>{t("task:noAgentProfileConfiguredDetail")}</p>
        {settingsWorkspaceId ? <SettingsLink workspaceId={settingsWorkspaceId} /> : null}
      </PopoverContent>
    </Popover>
  );
}

function SettingsLink({ workspaceId, className }: { workspaceId: string; className?: string }) {
  const { t } = useTranslation();
  return (
    <Link
      href={`/settings/workspaces/${workspaceId}`}
      className={className ?? "cursor-pointer text-primary underline underline-offset-2"}
      data-testid="sessionless-workspace-settings"
    >
      {t("task:openWorkspaceSettings")}
    </Link>
  );
}
