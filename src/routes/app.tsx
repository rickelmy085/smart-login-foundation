import { Outlet, createFileRoute, useRouterState } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { AppHeader } from "@/components/app-header";
import { AppSidebar } from "@/components/app-sidebar";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { useSession } from "@/lib/use-session";

export const Route = createFileRoute("/app")({
  component: AppLayout,
});

const titles: Record<string, string> = {
  "/app": "Dashboard",
  "/app/copilot": "Copilot",
  "/app/historico": "Histórico de consultas",
  "/app/perfil": "Meu perfil",
};

function AppLayout() {
  const { session, ready, signOut } = useSession();
  const pathname = useRouterState({ select: (s) => s.location.pathname.replace(/\/$/, "") });

  if (!ready || !session) {
    return (
      <div className="grid min-h-svh place-items-center bg-background" role="status">
        <Loader2 className="size-6 animate-spin text-muted-foreground" aria-hidden="true" />
        <span className="sr-only">Carregando sessão…</span>
      </div>
    );
  }

  return (
    <SidebarProvider>
      <AppSidebar session={session} onSignOut={signOut} />
      <SidebarInset className="min-w-0">
        <AppHeader title={titles[pathname] ?? "B-Smart Copilot"} session={session} onSignOut={signOut} />
        <main className="flex-1 p-4 sm:p-6 lg:p-8">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
