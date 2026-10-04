import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import type { BackendContext } from "../fixtures/backend";
import type { ApiClient } from "./api-client";
import { makeGitEnv } from "./git-helper";

/** Env that turns the repository cleanup runtime flag on for a spec's backend. */
export const REPOSITORY_CLEANUP_ENV = { KANDEV_FEATURES_REPOSITORY_CLEANUP: "true" };

export const CLEANUP_TASK_TITLE_PREFIX = "Clean up repository: ";

export type CleanupRepository = {
  repositoryId: string;
  repoName: string;
  cleanup: () => Promise<void>;
};

/** Creates a one-commit git repository and registers it as a local repository. */
export async function createCleanupRepository(options: {
  backend: BackendContext;
  apiClient: ApiClient;
  workspaceId: string;
  name: string;
}): Promise<CleanupRepository> {
  const { backend, apiClient, workspaceId, name } = options;
  const repoDir = path.join(backend.tmpDir, "repos", name);
  fs.mkdirSync(repoDir, { recursive: true });
  fs.writeFileSync(path.join(repoDir, "README.md"), `# ${name}\n`);
  const gitEnv = makeGitEnv(backend.tmpDir);
  execSync("git init -b main", { cwd: repoDir, env: gitEnv });
  execSync("git add -A", { cwd: repoDir, env: gitEnv });
  execSync('git commit -m "initial"', { cwd: repoDir, env: gitEnv });
  const { id } = await apiClient.createRepository(workspaceId, repoDir, "main", { name });
  return {
    repositoryId: id,
    repoName: name,
    cleanup: async () => {
      await apiClient.rawRequest("DELETE", `/api/v1/repositories/${id}`).catch(() => undefined);
      fs.rmSync(repoDir, { recursive: true, force: true });
    },
  };
}

/** Number of cleanup tasks created for the repository name in the workspace. */
export async function countCleanupTasks(
  apiClient: ApiClient,
  workspaceId: string,
  repoName: string,
): Promise<number> {
  const { tasks } = await apiClient.listTasks(workspaceId);
  return tasks.filter((task) => task.title === `${CLEANUP_TASK_TITLE_PREFIX}${repoName}`).length;
}
