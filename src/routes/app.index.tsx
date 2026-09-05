import { Link, createFileRoute } from "@tanstack/react-router";
import { ArrowRight, Bot, Clock, FileText, MessageSquareText, Sparkles, TrendingUp } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getSession } from "@/lib/auth";
import { historyItems, statusLabels, suggestedPrompts, topicColors, weeklyActivity } from "@/lib/mock-data";
import { useEffect, useState } from "react";

export const Route = createFileRoute("/app/")({
  head: () => ({
    meta: [
      { title: "Dashboard — B-Smart Copilot" },
      { name: "description", content: "Visão geral das suas consultas e atalhos do B-Smart Copilot." },
      { property: "og:title", content: "Dashboard — B-Smart Copilot" },
      { property: "og:description", content: "Visão geral das suas consultas e atalhos do B-Smart Copilot." },
    ],
  }),
  component: DashboardPage,
});

const stats = [
  { label: "Consultas na semana", value: "32", delta: "+18% vs. semana anterior", icon: MessageSquareText },
  { label: "Tempo médio de resposta", value: "2,4s", delta: "Dentro da meta", icon: Clock },
  { label: "Documentos consultados", value: "18", delta: "4 novos nesta semana", icon: FileText },
  { label: "Taxa de resolução", value: "94%", delta: "+3 p.p.", icon: TrendingUp },
];

function greeting() {
  const h = new Date().getHours();
  if (h < 12) return "Bom dia";
  if (h < 18) return "Boa tarde";
  return "Boa noite";
}

function DashboardPage() {
  const [firstName, setFirstName] = useState("");
  useEffect(() => setFirstName(getSession()?.name.split(" ")[0] ?? ""), []);
  const max = Math.max(...weeklyActivity.map((d) => d.consultas));
  const recent = historyItems.slice(0, 4);

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Welcome */}
      <section className="relative overflow-hidden rounded-2xl bg-gradient-hero p-6 text-primary-foreground shadow-elegant sm:p-8">
        <div className="grid-overlay pointer-events-none absolute inset-0" aria-hidden="true" />
        <div className="relative flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div className="max-w-xl">
            <p className="text-sm font-medium text-primary-foreground/70">
              {greeting()}
              {firstName ? `, ${firstName}` : ""} 👋
            </p>
            <h2 className="mt-1 text-2xl font-bold sm:text-3xl">
              O que você precisa resolver hoje?
            </h2>
            <p className="mt-2 text-sm text-primary-foreground/75">
              Seu copiloto está pronto para consultar políticas, procedimentos e documentos internos.
            </p>
          </div>
          <Button asChild size="lg" variant="secondary" className="shrink-0 bg-brand text-brand-foreground hover:bg-brand/90">
            <Link to="/app/copilot">
              <Sparkles aria-hidden="true" /> Abrir o Copilot
            </Link>
          </Button>
        </div>
      </section>

      {/* Stats */}
      <section aria-label="Indicadores" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map(({ label, value, delta, icon: Icon }) => (
          <Card key={label} className="shadow-panel">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardDescription>{label}</CardDescription>
              <span className="grid size-8 place-items-center rounded-lg bg-accent text-accent-foreground">
                <Icon className="size-4" aria-hidden="true" />
              </span>
            </CardHeader>
            <CardContent>
              <p className="font-display text-3xl font-bold tracking-tight">{value}</p>
              <p className="mt-1 text-xs text-muted-foreground">{delta}</p>
            </CardContent>
          </Card>
        ))}
      </section>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Activity */}
        <Card className="shadow-panel lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-base">Atividade semanal</CardTitle>
            <CardDescription>Consultas realizadas por dia</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="flex h-40 items-end gap-2" aria-label="Gráfico de consultas por dia">
              {weeklyActivity.map((d) => (
                <li key={d.day} className="flex flex-1 flex-col items-center gap-2">
                  <div className="flex w-full flex-1 items-end">
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
          </CardContent>
        </Card>

        {/* Recent */}
        <Card className="shadow-panel lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle className="text-base">Consultas recentes</CardTitle>
              <CardDescription>Suas últimas interações com o Copilot</CardDescription>
            </div>
            <Button asChild variant="ghost" size="sm">
              <Link to="/app/historico">
                Ver tudo <ArrowRight aria-hidden="true" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            <ul className="divide-y divide-border">
              {recent.map((item) => (
                <li key={item.id} className="flex items-start gap-3 py-3 first:pt-0 last:pb-0">
                  <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg bg-muted">
                    <Bot className="size-4 text-muted-foreground" aria-hidden="true" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{item.title}</p>
                    <p className="truncate text-xs text-muted-foreground">{item.preview}</p>
                  </div>
                  <div className="hidden shrink-0 flex-col items-end gap-1 sm:flex">
                    <Badge variant="outline" className={topicColors[item.topic]}>
                      {item.topic}
                    </Badge>
                    <span className="text-[11px] text-muted-foreground">{statusLabels[item.status]}</span>
                  </div>
                </li>
              ))}
            </ul>
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
              <Sparkles className="mb-2 size-4 text-brand" aria-hidden="true" />
              <span className="text-card-foreground">{p}</span>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}
