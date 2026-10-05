import { useCallback, useEffect, useRef, useState } from "react";
import { useAppStore } from "@/components/state-provider";
import { useFeature } from "@/hooks/domains/features/use-feature";
import { useForegroundRefresh } from "@/hooks/use-foreground-refresh";
import {
  getWaitingOwner,
  type WaitingOwnerFailedWorkspace,
  type WaitingOwnerItem,
} from "@/lib/api/domains/plan-files-api";
import { getWebSocketClient } from "@/lib/ws/connection";

// A task event can mean a plan entered or left the waiting step. Events come
// in bursts while agents work, so they share one trailing read.
const TASK_EVENT_COALESCE_MS = 2000;
const TASK_EVENTS = ["task.created", "task.updated", "task.state_changed", "task.deleted"] as const;

export type WaitingOwnerState = {
  status: "loading" | "ready" | "error";
  items: WaitingOwnerItem[];
  failedWorkspaces: WaitingOwnerFailedWorkspace[];
  reload: () => void;
};

/**
 * Plans waiting for the owner across every accessible workspace, read while the
 * planFiles feature is on. A failed read keeps the last list and reports the
 * error. The list is re-read when the window regains focus and, coalesced, on
 * task events.
 */
export function useWaitingOwner(): WaitingOwnerState {
  const enabled = useFeature("planFiles");
  const connectionStatus = useAppStore((s) => s.connection.status);
  const [status, setStatus] = useState<WaitingOwnerState["status"]>("loading");
  const [items, setItems] = useState<WaitingOwnerItem[]>([]);
  const [failedWorkspaces, setFailedWorkspaces] = useState<WaitingOwnerFailedWorkspace[]>([]);
  const latestRead = useRef(0);

  const reload = useCallback(() => {
    const read = ++latestRead.current;
    getWaitingOwner()
      .then((result) => {
        if (read !== latestRead.current) return;
        setItems(result.items);
        setFailedWorkspaces(result.failed_workspaces);
        setStatus("ready");
      })
      .catch(() => {
        if (read === latestRead.current) setStatus("error");
      });
  }, []);

  useEffect(() => {
    if (!enabled) return;
    reload();
    return () => {
      latestRead.current++;
    };
  }, [enabled, reload]);

  useForegroundRefresh(reload, enabled);
  useTaskEventReload(enabled, connectionStatus, reload);

  return { status, items, failedWorkspaces, reload };
}

function useTaskEventReload(enabled: boolean, connectionStatus: string, reload: () => void) {
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => {
    if (!enabled) return;
    const client = getWebSocketClient();
    if (!client) return;
    const schedule = () => {
      if (timer.current !== undefined) return;
      timer.current = window.setTimeout(() => {
        timer.current = undefined;
        reload();
      }, TASK_EVENT_COALESCE_MS);
    };
    const offs = TASK_EVENTS.map((type) => client.on(type, schedule));
    return () => offs.forEach((off) => off());
  }, [enabled, connectionStatus, reload]);

  useEffect(
    () => () => {
      if (timer.current !== undefined) window.clearTimeout(timer.current);
    },
    [],
  );
}
