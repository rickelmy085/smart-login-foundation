import { Link, createFileRoute } from "@tanstack/react-router";
import { ArrowRight, Bot, Clock, FileText, MessageSquareText, Sparkles, TrendingUp } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiFetch } from "@/lib/api";
import { getSession } from "@/lib/auth";
import { suggestedPrompts } from "@/lib/mock-data";
import { useEffect, useState } from "react";

export const Route = createFileRoute("/app/")({
  head: () => ({
    meta: [
      { title: "Dashboard — ABIS" },
      { name: "description", content: "Visão geral das suas consultas e atalhos do ABIS." },
      { property: "og:title", content: "Dashboard — ABIS" },
      { property: "og:description", content: "Visão geral das suas consultas e atalhos do ABIS." },
    ],
  }),
  component: DashboardPage,
});

function greeting() {
  const h = new Date().getHours();
  if (h < 12) return "Bom dia";
  if (h < 18) return "Boa tarde";
  return "Boa noite";
}

type HistoryItem = {
  id: string;
  type: string;
  title: string;
  preview: string;
  status: string;
  createdAt: string;
  sources: number;
};

type StatState = "loading" | "error" | "success";

const statDefinitions = [
  { key: "queries", label: "Consultas na semana", icon: MessageSquareText },
  { key: "avgTime", label: "Tempo médio de resposta", icon: Clock },
  { key: "documents", label: "Documentos gerados", icon: FileText },
  { key: "resolution", label: "Taxa de resolução", icon: TrendingUp },
];

function DashboardPage() {
  const [fullName, setFullName] = useState("");
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [statsState, setStatsState] = useState<StatState>("loading");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getSession().then((s) => setFullName(s?.name ?? ""));
  }, []);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setStatsState("loading");
      setError(null);
      try {
        const res = await apiFetch("/api/history");
        const data = (await res.json()) as { history: HistoryItem[] };
        if (!cancelled) {
          setHistory(data.history ?? []);
          setStatsState("success");
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Erro ao carregar dados.");
          setStatsState("error");
        }
      }
    };
    load();
    return () => { cancelled = true; };
  }, []);

  const firstName = fullName.split(" ")[0] ?? "";

  const oneWeekAgo = Date.now() - 7 * 24 * 60 * 60 * 1000;

  const queriesThisWeek = history.filter(
    (h) => h.type === "chat" && new Date(h.createdAt).getTime() > oneWeekAgo,
  ).length;

  const documentsGenerated = history.filter((h) => h.type === "document").length;

  const completedCount = history.filter(
    (h) => h.status === "completed" || h.status === "generated" || h.status === "concluida",
  ).length;
  const totalCount = history.length;
  const resolutionRate = totalCount > 0 ? Math.round((completedCount / totalCount) * 100) : 0;

  const weeklyActivity = computeWeeklyActivity(history);

  const statValues: Record<string, { value: string; delta: string }> = {
    queries: {
      value: queriesThisWeek.toString(),
      delta: "consultas nos últimos 7 dias",
    },
    avgTime: { value: "N/A", delta: "não disponível" },
    documents: { value: documentsGenerated.toString(), delta: "documentos gerados" },
    resolution: {
      value: totalCount > 0 ? `${resolutionRate}%` : "N/A",
      delta: totalCount > 0 ? `${completedCount}/${totalCount} concluídas` : "sem dados",
    },
  };

  const max = weeklyActivity.length > 0 ? Math.max(...weeklyActivity.map((d) => d.consultas)) : 0;

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Welcome */}
      <section className="relative overflow-hidden rounded-2xl bg-gradient-hero p-6 text-hero-foreground shadow-elegant sm:p-8">
        <div className="grid-overlay pointer-events-none absolute inset-0" aria-hidden="true" />
        <div className="relative flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div className="max-w-xl">
            <p className="text-sm font-medium text-hero-foreground/70">
              {greeting()}
              {firstName ? `, ${firstName}` : ""} 👋
            </p>
            <h2 className="mt-1 text-2xl font-bold sm:text-3xl">
              O que você precisa resolver hoje?
            </h2>
            <p className="mt-2 text-sm text-hero-foreground/75">
              Consulte políticas, normativos e conhecimento operacional no ABIS.
            </p>
          </div>
          <Button asChild size="lg" variant="secondary" className="shrink-0 bg-brand text-brand-foreground hover:bg-brand/90">
            <Link to="/app/copilot">
              <Sparkles aria-hidden={true} /> Abrir o ABIS
            </Link>
          </Button>
        </div>
      </section>

      {/* Stats */}
      <section aria-label="Indicadores" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {statDefinitions.map(({ key, label, icon: Icon }) => {
          const stat = statValues[key];
          return (
            <Card key={key} className="shadow-panel">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardDescription>{label}</CardDescription>
                <span className="grid size-8 place-items-center rounded-lg bg-accent text-accent-foreground">
                  <Icon className="size-4" aria-hidden={true} />
                </span>
              </CardHeader>
              <CardContent>
                {statsState === "loading" ? (
                  <Skeleton className="h-8 w-3/4" />
                ) : statsState === "error" ? (
                  <p className="font-display text-3xl font-bold text-muted-foreground">--</p>
                ) : (
                  <>
                    <p className="font-display text-3xl font-bold tracking-tight">{stat.value}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{stat.delta}</p>
                  </>
                )}
              </CardContent>
            </Card>
          );
        })}
      </section>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Activity */}
        <Card className="shadow-panel lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-base">Atividade semanal</CardTitle>
            <CardDescription>Consultas realizadas por dia</CardDescription>
          </CardHeader>
          <CardContent>
            {statsState === "loading" ? (
              <Skeleton className="h-28 w-full" />
            ) : weeklyActivity.length === 0 || weeklyActivity.every((d) => d.consultas === 0) ? (
              <p className="py-6 text-center text-sm text-muted-foreground">Sem dados de atividade.</p>
            ) : (
              <ul className="flex items-end gap-2" aria-label="Gráfico de consultas por dia">
                {weeklyActivity.map((d) => (
                  <li key={d.day} className="flex flex-1 flex-col items-center gap-2">
                    <div className="flex h-28 w-full items-end">
                      <div
                        className="w-full rounded-t-md bg-primary/85 transition-[height] duration-500"
                        style={{ height: `${max ? (d.consultas / max) * 100 : 0}%`, minHeight: d.consultas ? 6 : 2 }}
                        role="img"
                        aria-label={`${d.day}: ${d.consultas} consultas`}
                      />
                    </div>
                    <span className="text-[11px] text-muted-foreground">{d.day}</span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        {/* Recent */}
        <Card className="shadow-panel lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle className="text-base">Interações recentes</CardTitle>
              <CardDescription>Suas últimas interações com o ABIS</CardDescription>
            </div>
            <Button asChild variant="ghost" size="sm">
              <Link to="/app/historico">
                Ver tudo <ArrowRight aria-hidden={true} />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            {statsState === "loading" ? (
              <div className="space-y-3">
                {Array.from({ length: 4 }).map((_, i) => (
                  <div key={i} className="flex items-start gap-3">
                    <Skeleton className="size-8 shrink-0 rounded-full" />
                    <div className="flex-1 space-y-1">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                  </div>
                ))}
              </div>
            ) : statsState === "error" ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                {error ?? "Não foi possível carregar o histórico."}
              </p>
            ) : history.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">Nenhuma interação registrada ainda.</p>
            ) : (
              <ul className="divide-y divide-border">
                {history.slice(0, 4).map((item) => (
                  <li key={item.id} className="flex items-start gap-3 py-3 first:pt-0 last:pb-0">
                    <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg bg-muted">
                      <Bot className="size-4 text-muted-foreground" aria-hidden={true} />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{item.title}</p>
                      <p className="truncate text-xs text-muted-foreground">{item.preview}</p>
                    </div>
                    <div className="hidden shrink-0 flex-col items-end gap-1 sm:flex">
                      <Badge variant="outline" className={statusVariant(item.status)}>
                        {statusLabel(item.status)}
                      </Badge>
                      <span className="text-[11px] text-muted-foreground">
                        {new Date(item.createdAt).toLocaleString("pt-BR")}
                      </span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Suggestions */}
      <section aria-labelledby="sugestoes">
        <h2 id="sugestoes" className="mb-3 text-base font-semibold">
          Sugestões para começar
        </h2>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {suggestedPrompts.map((p) => (
            <Link
              key={p}
              to="/app/copilot"
              className="group rounded-xl border border-border bg-card p-4 text-sm shadow-panel transition-colors hover:border-primary/40 hover:bg-accent/40"
            >
              <Sparkles className="mb-2 size-4 text-brand" aria-hidden={true} />
              <span className="text-card-foreground">{p}</span>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}

function computeWeeklyActivity(history: HistoryItem[]) {
  const days = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];
  const now = new Date();
  const counts = Array(7).fill(0);

  for (const item of history) {
    if (item.type !== "chat") continue;
    const d = new Date(item.createdAt);
    const diff = now.getTime() - d.getTime();
    const daysAgo = Math.floor(diff / (24 * 60 * 60 * 1000));
    if (daysAgo >= 0 && daysAgo < 7) {
      const dayIdx = (now.getDay() - daysAgo + 7) % 7;
      counts[dayIdx]++;
    }
  }

  const todayIdx = now.getDay();
  return days.map((day, i) => {
    const idx = (todayIdx + i) % 7;
    return { day, consultas: counts[idx] };
  });
}

function statusVariant(status: string) {
  if (status === "generated" || status === "completed") return "bg-success/15 text-success border-success/30";
  if (status === "blocked" || status === "needs_review") return "bg-destructive/15 text-destructive border-destructive/30";
  if (status === "ready_to_generate" || status === "collecting_data" || status === "validating") return "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40";
  return "bg-muted text-muted-foreground";
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    detected: "Detectada",
    collecting_data: "Coletando dados",
    validating: "Validando",
    ready_to_generate: "Pronta para gerar",
    generating: "Gerando",
    generated: "Gerada",
    needs_review: "Revisão",
    completed: "Concluída",
    blocked: "Bloqueada",
  };
  return map[status] ?? status;
}
