import { Link } from "@tanstack/react-router";
import { Bell, LogOut, Search, UserRound } from "lucide-react";

import { ThemeToggle } from "@/components/theme-toggle";
import { initials } from "@/components/app-sidebar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Session } from "@/lib/auth";
import { toast } from "sonner";

export function AppHeader({
  title,
  session,
  onSignOut,
}: {
  title: string;
  session: Session;
  onSignOut: () => void;
}) {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-border bg-background/80 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/70 sm:px-6">
      <SidebarTrigger aria-label="Alternar menu lateral" className="-ml-1" />
      <h1 className="truncate text-base font-semibold sm:text-lg">{title}</h1>

      <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
        <div className="relative hidden md:block">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <input
            type="search"
            aria-label="Buscar consultas"
            placeholder="Buscar consultas…"
            className="h-9 w-56 rounded-full border border-input bg-muted/40 pl-9 pr-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring lg:w-72"
            onKeyDown={(e) => {
              if (e.key === "Enter") toast.info("Busca global disponível em uma próxima etapa.");
            }}
          />
        </div>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="relative rounded-full"
              aria-label="Notificações (2 novas)"
              onClick={() => toast.info("Nenhuma notificação pendente no ambiente de demonstração.")}
            >
              <Bell />
              <span
                className="absolute right-2 top-2 size-2 rounded-full bg-brand ring-2 ring-background"
                aria-hidden="true"
              />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Notificações</TooltipContent>
        </Tooltip>

        <ThemeToggle />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="h-9 gap-2 rounded-full pl-1 pr-3"
              aria-label="Menu do usuário"
            >
              <span className="grid size-7 place-items-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
                {initials(session.name)}
              </span>
              <span className="hidden text-sm font-medium sm:inline">
                {session.name.split(" ")[0]}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="font-normal">
              <p className="text-sm font-semibold">{session.name}</p>
              <p className="text-xs text-muted-foreground">
                {session.role} · RE {session.re}
              </p>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link to="/app/perfil">
                <UserRound /> Meu perfil
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onSignOut} className="text-destructive focus:text-destructive">
              <LogOut /> Sair
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
