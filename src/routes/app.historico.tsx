import { createFileRoute } from "@tanstack/react-router";
import { Bot, FileText, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { historyItems, statusLabels, topicColors, type QueryStatus } from "@/lib/mock-data";

export const Route = createFileRoute("/app/historico")({
  head: () => ({
    meta: [
      { title: "Histórico — ABIS" },
      { name: "description", content: "Histórico visual das suas consultas ao ABIS." },
      { property: "og:title", content: "Histórico — ABIS" },
      { property: "og:description", content: "Histórico visual das suas consultas ao ABIS." },
    ],
  }),
  component: HistoryPage,
});

type Filter = "todas" | QueryStatus;

const dateFmt = new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short" });
const timeFmt = new Intl.DateTimeFormat("pt-BR", { hour: "2-digit", minute: "2-digit" });
const dayFmt = new Intl.DateTimeFormat("pt-BR", { weekday: "long", day: "numeric", month: "long" });

function statusVariant(s: QueryStatus) {
  if (s === "concluida") return "bg-success/15 text-success border-success/30";
  if (s === "em_andamento") return "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40";
  return "bg-muted text-muted-foreground";
}

function HistoryPage() {
  const [filter, setFilter] = useState<Filter>("todas");
  const [query, setQuery] = useState("");

  const groups = useMemo(() => {
    const q = query.toLowerCase();
    const list = historyItems.filter(
      (i) =>
        (filter === "todas" || i.status === filter) &&
        (!q || i.title.toLowerCase().includes(q) || i.preview.toLowerCase().includes(q)),
    );
    const map = new Map<string, typeof list>();
    for (const item of list) {
      const key = dayFmt.format(new Date(item.createdAt));
      map.set(key, [...(map.get(key) ?? []), item]);
    }
    return [...map.entries()];
  }, [filter, query]);

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Tabs value={filter} onValueChange={(v) => setFilter(v as Filter)}>
          <TabsList aria-label="Filtrar por status">
            <TabsTrigger value="todas">Todas</TabsTrigger>
            <TabsTrigger value="concluida">Concluídas</TabsTrigger>
            <TabsTrigger value="em_andamento">Em andamento</TabsTrigger>
            <TabsTrigger value="arquivada">Arquivadas</TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="relative sm:w-72">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <Input
            type="search"
            aria-label="Buscar no histórico"
            placeholder="Buscar no histórico…"
            className="pl-9"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      {groups.length === 0 ? (
        <Card className="shadow-panel">
          <CardContent className="py-12 text-center text-sm text-muted-foreground">
            Nenhuma consulta encontrada com esses filtros.
          </CardContent>
        </Card>
      ) : (
        groups.map(([day, items]) => (
          <section key={day} aria-labelledby={`day-${day}`}>
            <h2 id={`day-${day}`} className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground first-letter:uppercase">
              {day}
            </h2>
            <ol className="relative space-y-3 border-l border-border pl-6">
              {items.map((item) => {
                const d = new Date(item.createdAt);
                return (
                  <li key={item.id} className="relative">
                    <span
                      className="absolute -left-[1.85rem] top-4 grid size-5 place-items-center rounded-full bg-background ring-1 ring-border"
                      aria-hidden="true"
                    >
                      <Bot className="size-3 text-primary" />
                    </span>
                    <Card className="shadow-panel transition-colors hover:border-primary/40">
                      <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start">
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="font-medium">{item.title}</p>
                            <Badge variant="outline" className={topicColors[item.topic]}>
                              {item.topic}
                            </Badge>
                          </div>
                          <p className="mt-1 text-sm text-muted-foreground">{item.preview}</p>
                          <p className="mt-2 flex items-center gap-1 text-xs text-muted-foreground">
                            <FileText className="size-3.5" aria-hidden="true" />
                            {item.sources} {item.sources === 1 ? "fonte" : "fontes"} · {item.id}
                          </p>
                        </div>
                        <div className="flex shrink-0 items-center gap-3 sm:flex-col sm:items-end">
                          <Badge variant="outline" className={statusVariant(item.status)}>
                            {statusLabels[item.status]}
                          </Badge>
                          <time dateTime={item.createdAt} className="text-xs text-muted-foreground">
                            {dateFmt.format(d)} · {timeFmt.format(d)}
                          </time>
                        </div>
                      </CardContent>
                    </Card>
                  </li>
                );
              })}
            </ol>
          </section>
        ))
      )}
    </div>
  );
}
