import { lazy, Suspense } from "react";
import { AuthRouteRedirect, RouteLoading } from "./spa-route-chrome";

const PlansWaitingPageClient = lazy(() =>
  import("@/app/plans-waiting/plans-waiting-page-client").then((mod) => ({
    default: mod.PlansWaitingPageClient,
  })),
);

export function PlansWaitingRoute({ enabled }: { enabled: boolean }) {
  if (!enabled) return <AuthRouteRedirect />;
  return (
    <Suspense fallback={<RouteLoading routeNameKey="planFiles:waitingTitle" />}>
      <PlansWaitingPageClient />
    </Suspense>
  );
}
