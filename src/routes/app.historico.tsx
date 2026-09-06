import { createFileRoute } from "@tanstack/react-router";
import { Bot, FileText, Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiFetch } from "@/lib/api";

export type HistoryItem = {
  id: string;
  title: string;
  preview: string;
  status: string;
  createdAt: string;
  sources: number;
};

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

type Filter = "todas" | "concluida" | "em_andamento" | "arquivada";

const dateFmt = new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short" });
const timeFmt = new Intl.DateTimeFormat("pt-BR", { hour: "2-digit", minute: "2-digit" });
const dayFmt = new Intl.DateTimeFormat("pt-BR", { weekday: "long", day: "numeric", month: "long" });

function statusVariant(s: string) {
  if (s === "generated" || s === "completed") return "bg-success/15 text-success border-success/30";
  if (s === "blocked" || s === "needs_review") return "bg-destructive/15 text-destructive border-destructive/30";
  if (s === "ready_to_generate" || s === "collecting_data" || s === "validating") return "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40";
  return "bg-muted text-muted-foreground";
}

function statusLabel(s: string) {
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
  return map[s] ?? s;
}

function HistoryPage() {
  const [items, setItems] = useState<HistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<Filter>("todas");
  const [query, setQuery] = useState("");

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const res = await apiFetch("/api/history");
        const data = (await res.json()) as { history: HistoryItem[] };
        if (!cancelled) setItems(data.history ?? []);
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Erro ao carregar histórico.";
        if (!cancelled) toast.error(msg);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, []);

  const groups = useMemo(() => {
    const q = query.toLowerCase();
    const list = items.filter((i) => {
      const matchesFilter =
        filter === "todas" ||
        i.status === filter ||
        (filter === "concluida" && (i.status === "completed" || i.status === "generated")) ||
        (filter === "em_andamento" && ["collecting_data", "validating", "ready_to_generate", "generating", "detected"].includes(i.status)) ||
        (filter === "arquivada" && (i.status === "blocked" || i.status === "needs_review"));
      const matchesQuery = !q || i.title.toLowerCase().includes(q) || i.preview.toLowerCase().includes(q);
      return matchesFilter && matchesQuery;
    });
    const map = new Map<string, typeof list>();
    for (const item of list) {
      const key = dayFmt.format(new Date(item.createdAt));
      map.set(key, [...(map.get(key) ?? []), item]);
    }
    return [...map.entries()];
  }, [items, filter, query]);

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

      {loading ? (
        <Card className="shadow-panel">
          <CardContent className="py-12 text-center text-sm text-muted-foreground">Carregando...</CardContent>
        </Card>
      ) : groups.length === 0 ? (
        <Card className="shadow-panel">
          <CardContent className="py-12 text-center text-sm text-muted-foreground">
            Nenhuma consulta encontrada com esses filtros.
          </CardContent>
        </Card>
      ) : (
        groups.map(([day, dayItems]) => (
          <section key={day} aria-labelledby={`day-${day}`}>
            <h2 id={`day-${day}`} className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground first-letter:uppercase">
              {day}
            </h2>
            <ol className="relative space-y-3 border-l border-border pl-6">
              {dayItems.map((item) => {
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
                            <Badge variant="outline" className={statusVariant(item.status)}>
                              {statusLabel(item.status)}
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
                            {statusLabel(item.status)}
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
