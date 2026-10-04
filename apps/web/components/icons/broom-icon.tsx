import type { SVGProps } from "react";

/**
 * Outline broom in the Tabler stroke style (24px grid, 2px stroke). The
 * installed Tabler release ships no broom, so the cleanup action draws its own.
 */
export function BroomIcon({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className={className}
      data-testid="broom-icon"
      {...props}
    >
      <path d="M20 3l-6.5 6.5" />
      <path d="M11 8l5 5l-4.5 7.5l-8 -8z" />
      <path d="M12 13.5l-3.5 3.5" />
    </svg>
  );
}
