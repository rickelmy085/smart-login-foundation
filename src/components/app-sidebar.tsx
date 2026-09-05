import { Link } from "@tanstack/react-router";
import { Bot, History, LayoutDashboard, LogOut, UserRound } from "lucide-react";

import { BrandLogo } from "@/components/brand-logo";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import type { Session } from "@/lib/auth";

const navItems = [
  { to: "/app", label: "Dashboard", icon: LayoutDashboard, exact: true },
  { to: "/app/copilot", label: "ABIS", icon: Bot, exact: false },
  { to: "/app/historico", label: "Histórico", icon: History, exact: false },
  { to: "/app/perfil", label: "Perfil", icon: UserRound, exact: false },
] as const;

export function AppSidebar({ session, onSignOut }: { session: Session; onSignOut: () => void }) {
  const { state, isMobile, setOpenMobile } = useSidebar();
  const collapsed = state === "collapsed" && !isMobile;

  return (
    <Sidebar collapsible="icon" aria-label="Navegação principal">
      <SidebarHeader className="h-16 justify-center border-b border-sidebar-border px-3">
        {collapsed ? (
          <div
            className="mx-auto grid size-8 place-items-center rounded-lg bg-brand font-display text-sm font-extrabold text-brand-foreground"
            aria-label="ABIS"
          >
            A
          </div>
        ) : (
          <BrandLogo size="sm" />
        )}
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Menu</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map(({ to, label, icon: Icon, exact }) => (
                <SidebarMenuItem key={to}>
                  <SidebarMenuButton asChild tooltip={label}>
                    <Link
                      to={to}
                      activeOptions={{ exact }}
                      activeProps={{
                        className: "bg-sidebar-accent text-sidebar-accent-foreground font-semibold",
                        "aria-current": "page",
                      }}
                      onClick={() => setOpenMobile(false)}
                    >
                      <Icon aria-hidden="true" />
                      <span>{label}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip={session.name}>
              <Link to="/app/perfil" onClick={() => setOpenMobile(false)}>
                <span className="grid size-6 shrink-0 place-items-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
                  {initials(session.name)}
                </span>
                <span className="flex min-w-0 flex-col leading-tight">
                  <span className="truncate text-sm font-medium">{session.name}</span>
                  <span className="truncate text-xs text-muted-foreground">RE {session.re}</span>
                </span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton onClick={onSignOut} tooltip="Sair">
              <LogOut aria-hidden="true" />
              <span>Sair</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

export function initials(name: string) {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((n) => n[0]!.toUpperCase())
    .join("");
}
