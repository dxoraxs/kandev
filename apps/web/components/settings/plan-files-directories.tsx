"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconPlus, IconX } from "@tabler/icons-react";
import { Button } from "@kandev/ui/button";
import { Input } from "@kandev/ui/input";
import { settingsActionClassName, settingsControlClassName } from "./settings-control";
import { SettingsFieldLabel } from "./settings-typography";

type DirectoriesProps = {
  directories: string[];
  onChange: (directories: string[]) => void;
};

/** Add/remove editor for the repository-relative directories that are scanned. */
export function PlanFilesDirectories({ directories, onChange }: DirectoriesProps) {
  const { t } = useTranslation();
  const [value, setValue] = useState("");
  const candidate = value.trim();
  const canAdd = candidate !== "" && !directories.includes(candidate);

  const add = () => {
    if (!canAdd) return;
    onChange([...directories, candidate]);
    setValue("");
  };

  return (
    <div className="space-y-2" data-testid="plan-files-directories">
      <SettingsFieldLabel htmlFor="plan-files-directory-input">
        {t("planFiles:directories")}
      </SettingsFieldLabel>
      <ul className="flex flex-col gap-1.5 sm:flex-row sm:flex-wrap">
        {directories.map((directory) => (
          <li
            key={directory}
            className="flex min-w-0 items-center justify-between gap-2 rounded-md border bg-muted/40 py-1 pl-2.5 pr-1 font-mono text-xs"
          >
            <span className="min-w-0 truncate">{directory}</span>
            <button
              type="button"
              aria-label={t("planFiles:removeDirectory", { directory })}
              onClick={() => onChange(directories.filter((item) => item !== directory))}
              className="flex size-6 shrink-0 cursor-pointer items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground max-md:size-11 [@media(pointer:coarse)]:size-11"
            >
              <IconX className="h-4 w-4" aria-hidden="true" />
            </button>
          </li>
        ))}
      </ul>
      <div className="flex flex-col gap-2 sm:flex-row">
        <Input
          id="plan-files-directory-input"
          value={value}
          placeholder={t("planFiles:directoryPlaceholder")}
          onChange={(event) => setValue(event.target.value)}
          onKeyDown={(event) => {
            if (event.key !== "Enter") return;
            event.preventDefault();
            add();
          }}
          className={settingsControlClassName("sm:max-w-xs")}
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!canAdd}
          onClick={add}
          className={settingsActionClassName("w-full cursor-pointer sm:w-auto")}
          data-testid="plan-files-add-directory"
        >
          <IconPlus className="mr-2 h-4 w-4" />
          {t("planFiles:addDirectory")}
        </Button>
      </div>
    </div>
  );
}
