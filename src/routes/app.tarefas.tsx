import { createFileRoute, Link } from "@tanstack/react-router";
import { Bot, Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { listTasks, createTask, type Task } from "@/lib/workflow";

export const Route = createFileRoute("/app/tarefas")({
  head: () => ({
    meta: [
      { title: "Tarefas — ABIS" },
      { name: "description", content: "Fluxo agentic de tarefas e geração de documentos." },
      { property: "og:title", content: "Tarefas — ABIS" },
      { property: "og:description", content: "Fluxo agentic de tarefas e geração de documentos." },
    ],
  }),
  component: TasksPage,
});

function statusVariant(status: Task["status"]) {
  if (status === "generated" || status === "completed") return "bg-success/15 text-success border-success/30";
  if (status === "blocked" || status === "needs_review") return "bg-destructive/15 text-destructive border-destructive/30";
  if (status === "ready_to_generate" || status === "collecting_data" || status === "validating") return "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40";
  return "bg-muted text-muted-foreground";
}

function statusLabel(status: Task["status"]) {
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

function TasksPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [question, setQuestion] = useState("");

  const load = async () => {
    setLoading(true);
    try {
      const data = await listTasks();
      setTasks(data.tasks);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Erro ao carregar tarefas.";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const q = question.trim();
    if (!q || creating) return;
    setCreating(true);
    try {
      await createTask(q);
      setQuestion("");
      toast.success("Tarefa criada com sucesso.");
      await load();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Erro ao criar tarefa.";
      toast.error(message);
    } finally {
      setCreating(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Tarefas</h2>
          <p className="text-sm text-muted-foreground">Fluxo agentic de solicitações e geração de documentos.</p>
        </div>
      </div>

      <Card className="shadow-panel">
        <CardHeader>
          <CardTitle className="text-base">Nova solicitação</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-3 sm:flex-row">
            <Textarea
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="Descreva sua solicitação, por exemplo: 'Preciso de uma solicitação de aquisição de TI'"
              rows={2}
              className="min-h-0 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
            />
            <div className="flex items-center gap-2 sm:self-end">
              <Button type="submit" size="icon" disabled={!question.trim() || creating} aria-label="Criar tarefa">
                <Plus />
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <div className="space-y-3">
        {loading ? (
          <Card className="shadow-panel">
            <CardContent className="py-12 text-center text-sm text-muted-foreground">Carregando...</CardContent>
          </Card>
        ) : tasks.length === 0 ? (
          <Card className="shadow-panel">
            <CardContent className="py-12 text-center text-sm text-muted-foreground">
              Nenhuma tarefa encontrada.
            </CardContent>
          </Card>
        ) : (
          tasks.map((t) => (
            <Link key={t.id} to={`/app/tarefas/${t.id}`} className="block">
              <Card className="shadow-panel transition-colors hover:border-primary/40">
                <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-medium">{t.originalRequest}</p>
                      <Badge variant="outline" className={statusVariant(t.status)}>
                        {statusLabel(t.status)}
                      </Badge>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {t.intent} · {new Date(t.createdAt).toLocaleString("pt-BR")}
                    </p>
                  </div>
                  <div className="hidden shrink-0 sm:block">
                    <Button variant="ghost" size="sm">
                      Abrir <Bot className="ml-2 size-4" aria-hidden="true" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))
        )}
      </div>
    </div>
  );
}
