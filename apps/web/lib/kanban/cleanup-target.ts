type CleanupRepository = { id: string; source_type?: string | null };

export type CleanupTarget<R extends CleanupRepository> =
  | { kind: "hidden" }
  | { kind: "needs_selection" }
  | { kind: "ready"; repository: R };

/**
 * Which repository the board's cleanup action works on: the repository
 * selected in the board filter when it is local, otherwise the workspace's
 * only local repository. Without a local repository there is no action.
 */
export function resolveCleanupTarget<R extends CleanupRepository>(
  repositories: R[],
  selectedRepositoryId: string | null,
): CleanupTarget<R> {
  const local = repositories.filter((repo) => repo.source_type === "local");
  if (local.length === 0) return { kind: "hidden" };
  const selected = repositories.find((repo) => repo.id === selectedRepositoryId);
  if (selected) {
    return selected.source_type === "local"
      ? { kind: "ready", repository: selected }
      : { kind: "needs_selection" };
  }
  if (local.length === 1) return { kind: "ready", repository: local[0] };
  return { kind: "needs_selection" };
}
