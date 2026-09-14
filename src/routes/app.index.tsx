import { Link, createFileRoute } from "@tanstack/react-router";
import {
  ArrowRight,
  Bot,
  Clock,
  FileText,
  MessageSquareText,
  Sparkles,
  CheckCircle,
  AlertTriangle,
  Target,
} from "lucide-react";
import { useEffect, useState, useMemo } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { getSession } from "@/lib/auth";
import { suggestedPrompts } from "@/lib/mock-data";
import { type HistoryItem, listHistory } from "@/lib/workflow";

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

type StatState = "loading" | "error" | "success";

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
        const res = await listHistory();
        if (!cancelled) {
          setHistory(res.history ?? []);
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
    return () => {
      cancelled = true;
    };
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

  const weeklyActivity = useMemo(() => computeWeeklyActivity(history), [history]);
  const maxActivity =
    weeklyActivity.length > 0 ? Math.max(...weeklyActivity.map((d) => d.consultas)) : 0;

  const pendingTasks = history.filter((h) =>
    ["collecting_data", "validating", "ready_to_generate", "generating", "detected"].includes(
      h.status,
    ),
  ).length;
  const needsReview = history.filter((h) =>
    ["needs_review", "blocked"].includes(h.status),
  ).length;

  return (
    <div className="mx-auto max-w-7xl space-y-6 pb-8">
      <section className="relative overflow-hidden rounded-2xl bg-gradient-hero p-6 text-hero-foreground shadow-elegant sm:p-8">
        <div className="grid-overlay pointer-events-none absolute inset-0" aria-hidden="true" />
        <div className="relative flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div className="max-w-xl">
            <p className="text-sm font-medium text-hero-foreground/70">
              {greeting()}
              {firstName ? `, ${firstName}` : ""}
            </p>
            <h2 className="mt-1 text-2xl font-bold sm:text-3xl">
              O que você precisa resolver hoje?
            </h2>
            <p className="mt-2 text-sm text-hero-foreground/75">
              Consulte políticas, normativos e conhecimento operacional no ABIS.
            </p>
          </div>
          <Button
            asChild
            size="lg"
            variant="secondary"
            className="shrink-0 bg-brand text-brand-foreground hover:bg-brand/90"
          >
            <Link to="/app/copilot">
              <Sparkles aria-hidden={true} /> Abrir o ABIS
            </Link>
          </Button>
        </div>
      </section>

      <section
        aria-label="Indicadores principais"
        className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4"
      >
        <MetricCard
          label="Consultas na semana"
          value={queriesThisWeek.toString()}
          delta="últimos 7 dias"
          icon={MessageSquareText}
          state={statsState}
        />
        <MetricCard
          label="Documentos gerados"
          value={documentsGenerated.toString()}
          delta="total acumulado"
          icon={FileText}
          state={statsState}
        />
        <MetricCard
          label="Taxa de conclusão"
          value={totalCount > 0 ? `${resolutionRate}%` : "—"}
          delta={totalCount > 0 ? `${completedCount} de ${totalCount}` : "sem dados"}
          icon={CheckCircle}
          state={statsState}
        />
        <MetricCard
          label="Tarefas ativas"
          value={pendingTasks.toString()}
          delta={needsReview > 0 ? `${needsReview} aguardando revisão` : "todas em dia"}
          icon={Target}
          state={statsState}
        />
      </section>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="shadow-panel lg:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-base">Atividade semanal</CardTitle>
                <CardDescription>Consultas realizadas por dia</CardDescription>
              </div>
              {statsState === "success" && history.length > 0 && (
                <Badge variant="outline" className="text-xs">
                  {history.length} interações no total
                </Badge>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {statsState === "loading" ? (
              <Skeleton className="h-28 w-full" />
            ) : weeklyActivity.length === 0 ||
              weeklyActivity.every((d) => d.consultas === 0) ? (
              <div className="py-8 text-center">
                <Bot
                  className="size-8 mx-auto text-muted-foreground/50"
                  aria-hidden="true"
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  Nenhuma atividade registrada ainda.
                </p>
                <p className="mt-1 text-xs text-muted-foreground/70">
                  Comece uma conversa no ABIS para ver a atividade aqui.
                </p>
              </div>
            ) : (
              <>
                <ul
                  className="flex items-end gap-2"
                  aria-label="Gráfico de barras de consultas por dia da semana"
                >
                  {weeklyActivity.map((d) => (
                    <li
                      key={d.day}
                      className="flex flex-1 flex-col items-center gap-2"
                    >
                      <div className="flex h-28 w-full items-end">
                        <div
                          className="w-full rounded-t-md bg-primary/85 transition-[height] duration-500 hover:bg-primary"
                          style={{
                            height: `${maxActivity ? (d.consultas / maxActivity) * 100 : 0}%`,
                            minHeight: d.consultas ? 6 : 2,
                          }}
                          role="img"
                          aria-label={`${d.day}: ${d.consultas} consultas`}
                        />
                      </div>
                      <span className="text-[11px] text-muted-foreground">{d.day}</span>
                    </li>
                  ))}
                </ul>
                <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
                  <span>{queriesThisWeek} consultas esta semana</span>
                  <span>{history.length} interações no total</span>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card className="shadow-panel">
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Sparkles className="text-brand" aria-hidden="true" />
                Ações rápidas
              </CardTitle>
              <CardDescription>Atalhos para tarefas comuns</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {suggestedPrompts.slice(0, 4).map((p) => (
                <Link
                  key={p}
                  to="/app/copilot"
                  className="group flex items-start gap-3 rounded-xl border border-border bg-card p-3 text-sm shadow-panel transition-colors hover:border-primary/40 hover:bg-accent/40"
                >
                  <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg bg-brand/10 text-brand">
                    <Sparkles className="size-4" aria-hidden="true" />
                  </span>
                  <span className="text-card-foreground group-hover:text-primary transition-colors">
                    {p}
                  </span>
                </Link>
              ))}
            </CardContent>
          </Card>

          <Card className="shadow-panel">
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Target className="text-green-600" aria-hidden="true" />
                Status das tarefas
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <StatusRow
                label="Concluídas"
                value={completedCount}
                color="text-green-600"
                icon={CheckCircle}
              />
              <StatusRow
                label="Em andamento"
                value={pendingTasks}
                color="text-blue-600"
                icon={Clock}
              />
              <StatusRow
                label="Precisam revisão"
                value={needsReview}
                color="text-yellow-600"
                icon={AlertTriangle}
              />
              <Separator />
              <StatusRow
                label="Total"
                value={history.length}
                color="text-muted-foreground"
                icon={FileText}
              />
            </CardContent>
          </Card>
        </div>
      </div>

      <section aria-labelledby="recent-title">
        <div className="mb-4 flex items-center justify-between">
          <h2 id="recent-title" className="text-base font-semibold">
            Interações recentes
          </h2>
          <Button asChild variant="ghost" size="sm">
            <Link to="/app/historico">
              Ver tudo <ArrowRight aria-hidden={true} />
            </Link>
          </Button>
        </div>

        {statsState === "loading" ? (
          <Card className="shadow-panel">
            <CardContent className="py-6">
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
            </CardContent>
          </Card>
        ) : statsState === "error" ? (
          <Card className="shadow-panel">
            <CardContent className="py-6 text-center text-sm text-muted-foreground">
              {error ?? "Não foi possível carregar o histórico."}
            </CardContent>
          </Card>
        ) : history.length === 0 ? (
          <Card className="shadow-panel">
            <CardContent className="py-12 text-center">
              <Bot
                className="size-12 mx-auto text-muted-foreground/50"
                aria-hidden="true"
              />
              <h3 className="mt-3 text-base font-medium">Nenhuma interação ainda</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Suas conversas e documentos aparecerão aqui.
              </p>
              <Button asChild className="mt-4" size="sm">
                <Link to="/app/copilot">
                  <Sparkles className="mr-2 size-4" aria-hidden="true" />
                  Começar agora
                </Link>
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {history.slice(0, 5).map((item) => (
              <RecentActivityItem key={item.id} item={item} />
            ))}
          </div>
        )}
      </section>

      <section aria-labelledby="sugestoes" className="pt-4">
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
              <Sparkles
                className="mb-2 size-4 text-brand"
                aria-hidden={true}
              />
              <span className="text-card-foreground group-hover:text-primary transition-colors">
                {p}
              </span>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}

function MetricCard({
  label,
  value,
  delta,
  icon: Icon,
  state,
}: {
  label: string;
  value: string;
  delta: string;
  icon: React.ComponentType<{ className?: string; ariaHidden?: boolean }>;
  state: "loading" | "error" | "success";
}) {
  return (
    <Card className="shadow-panel transition-shadow hover:shadow-elegant">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardDescription>{label}</CardDescription>
        <span className="grid size-8 place-items-center rounded-lg bg-accent text-accent-foreground">
          <Icon className="size-4" aria-hidden={true} />
        </span>
      </CardHeader>
      <CardContent>
        {state === "loading" ? (
          <Skeleton className="h-8 w-3/4" />
        ) : state === "error" ? (
          <p className="font-display text-3xl font-bold text-muted-foreground">—</p>
        ) : (
          <>
            <p className="font-display text-3xl font-bold tracking-tight">{value}</p>
            <p className="mt-1 text-xs text-muted-foreground">{delta}</p>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function StatusRow({
  label,
  value,
  color,
  icon: Icon,
}: {
  label: string;
  value: number;
  color: string;
  icon: React.ComponentType<{ className?: string; ariaHidden?: boolean }>;
}) {
  return (
    <div className="flex items-center justify-between py-1">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Icon className={`size-4 ${color}`} aria-hidden={true} />
        <span>{label}</span>
      </div>
      <span className="font-mono font-medium text-base">{value}</span>
    </div>
  );
}

function RecentActivityItem({ item }: { item: HistoryItem }) {
  const statusColors: Record<string, string> = {
    generated: "bg-success/15 text-success border-success/30",
    completed: "bg-success/15 text-success border-success/30",
    concluida: "bg-success/15 text-success border-success/30",
    blocked: "bg-destructive/15 text-destructive border-destructive/30",
    needs_review: "bg-destructive/15 text-destructive border-destructive/30",
    ready_to_generate:
      "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40",
    collecting_data:
      "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40",
    validating: "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40",
    generating: "bg-primary/15 text-primary border-primary/30",
    detected: "bg-muted text-muted-foreground",
  };

  const statusLabels: Record<string, string> = {
    detected: "Detectada",
    collecting_data: "Coletando dados",
    validating: "Validando",
    ready_to_generate: "Pronta para gerar",
    generating: "Gerando",
    generated: "Gerada",
    needs_review: "Revisão",
    completed: "Concluída",
    concluida: "Concluída",
    blocked: "Bloqueada",
  };

  return (
    <Link to={`/app/tarefas/${item.id}`} className="block">
      <Card className="shadow-panel transition-colors hover:border-primary/40 hover:shadow-elegant">
        <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <p className="font-medium truncate">{item.title}</p>
              <Badge
                variant="outline"
                className={statusColors[item.status] || "bg-muted text-muted-foreground"}
              >
                {statusLabels[item.status] || item.status}
              </Badge>
            </div>
            <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{item.preview}</p>
          </div>
          <div className="hidden shrink-0 sm:block text-right">
            <p className="text-[11px] text-muted-foreground">
              {new Date(item.createdAt).toLocaleString("pt-BR")}
            </p>
          </div>
        </CardContent>
      </Card>
    </Link>
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
