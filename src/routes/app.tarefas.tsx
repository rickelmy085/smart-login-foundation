import { createFileRoute, Link } from "@tanstack/react-router";
import { Bot, Plus, Calendar, Flag, CheckCircle2, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { format, isPast, parseISO } from "date-fns";
import { ptBR } from "date-fns/locale";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { DatePicker } from "@/components/ui/date-picker";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  listTasks,
  createTask,
  updateTaskStatus,
  deleteTask,
  type Task,
  type TaskPriority,
} from "@/lib/workflow";

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
  if (status === "generated" || status === "completed")
    return "bg-success/15 text-success border-success/30";
  if (status === "blocked" || status === "needs_review")
    return "bg-destructive/15 text-destructive border-destructive/30";
  if (status === "ready_to_generate" || status === "collecting_data" || status === "validating")
    return "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40";
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

function priorityVariant(priority: TaskPriority | undefined) {
  if (priority === "urgente") return "bg-destructive/15 text-destructive border-destructive/30";
  if (priority === "alta")
    return "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40";
  if (priority === "media") return "bg-blue/15 text-blue-foreground border-blue/30";
  if (priority === "baixa") return "bg-muted text-muted-foreground";
  return "bg-muted text-muted-foreground";
}

function priorityLabel(priority: TaskPriority | undefined) {
  const map: Record<string, string> = {
    baixa: "Baixa",
    media: "Média",
    alta: "Alta",
    urgente: "Urgente",
  };
  return map[priority ?? ""] ?? priority ?? "—";
}

function formatDeadline(deadline: string | undefined) {
  if (!deadline) return null;
  try {
    const date = parseISO(deadline);
    return format(date, "dd/MM/yyyy", { locale: ptBR });
  } catch {
    return deadline;
  }
}

function isDeadlineOverdue(deadline: string | undefined) {
  if (!deadline) return false;
  try {
    const date = parseISO(deadline);
    return isPast(date);
  } catch {
    return false;
  }
}

function TasksPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [question, setQuestion] = useState("");
  const [deadline, setDeadline] = useState<Date | null>(null);
  const [priority, setPriority] = useState<TaskPriority>("media");

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
      const deadlineStr = deadline ? deadline.toISOString().split("T")[0] : undefined;
      await createTask(q, deadlineStr, priority);
      setQuestion("");
      setDeadline(null);
      setPriority("media");
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
          <p className="text-sm text-muted-foreground">
            Fluxo agentic de solicitações e geração de documentos.
          </p>
        </div>
      </div>

      <Card className="shadow-panel">
        <CardHeader>
          <CardTitle className="text-base">Nova solicitação</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Textarea
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Descreva sua solicitação, por exemplo: 'Preciso de uma solicitação de aquisição de TI'"
                rows={2}
                className="min-h-0 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
              />
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex items-center gap-2">
                  <Calendar className="size-4 text-muted-foreground" aria-hidden="true" />
                  <DatePicker
                    value={deadline}
                    onChange={setDeadline}
                    placeholder="Data de prazo"
                    className="w-[200px]"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <Flag className="size-4 text-muted-foreground" aria-hidden="true" />
                  <Select value={priority} onValueChange={setPriority}>
                    <SelectTrigger className="w-[160px]">
                      <SelectValue placeholder="Prioridade" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="baixa">Baixa</SelectItem>
                      <SelectItem value="media">Média</SelectItem>
                      <SelectItem value="alta">Alta</SelectItem>
                      <SelectItem value="urgente">Urgente</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-center gap-2 ml-auto">
                  <Button
                    type="submit"
                    size="icon"
                    disabled={!question.trim() || creating}
                    aria-label="Criar tarefa"
                  >
                    <Plus />
                  </Button>
                </div>
              </div>
            </div>
          </form>
        </CardContent>
      </Card>

      <div className="space-y-3">
        {loading ? (
          <Card className="shadow-panel">
            <CardContent className="py-12 text-center text-sm text-muted-foreground">
              Carregando...
            </CardContent>
          </Card>
        ) : tasks.length === 0 ? (
          <Card className="shadow-panel">
            <CardContent className="py-12 text-center text-sm text-muted-foreground">
              Nenhuma tarefa encontrada.
            </CardContent>
          </Card>
        ) : (
          tasks.map((t) => (
            <Card className="shadow-panel transition-colors hover:border-primary/40">
              <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-medium">{t.originalRequest}</p>
                    <Badge variant="outline" className={statusVariant(t.status)}>
                      {statusLabel(t.status)}
                    </Badge>
                    {t.priority && (
                      <Badge variant="outline" className={priorityVariant(t.priority)}>
                        {priorityLabel(t.priority)}
                      </Badge>
                    )}
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t.intent} · {new Date(t.createdAt).toLocaleString("pt-BR")}
                  </p>
                  {t.deadline && (
                    <p
                      className="mt-1 flex items-center gap-1 text-xs"
                      style={{
                        color:
                          isDeadlineOverdue(t.deadline) && t.status !== "completed"
                            ? "var(--destructive)"
                            : "var(--muted-foreground)",
                      }}
                    >
                      <Calendar className="size-3" aria-hidden="true" />
                      <span>Prazo: {formatDeadline(t.deadline)}</span>
                      {isDeadlineOverdue(t.deadline) && t.status !== "completed" && (
                        <span className="font-medium">(Atrasada)</span>
                      )}
                    </p>
                  )}
                </div>
                <div className="hidden shrink-0 sm:block flex items-center gap-2">
                  {t.status !== "completed" && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        try {
                          await updateTaskStatus(t.id, "completed");
                          toast.success("Tarefa marcada como concluída.");
                          await load();
                        } catch (err) {
                          const message =
                            err instanceof Error ? err.message : "Erro ao concluir tarefa.";
                          toast.error(message);
                        }
                      }}
                    >
                      <CheckCircle2 className="mr-2 size-4" aria-hidden="true" />
                      Concluir
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                    onClick={async (e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      if (
                        !confirm(
                          "Tem certeza que deseja excluir esta tarefa? Esta ação não pode ser desfeita.",
                        )
                      ) {
                        return;
                      }
                      try {
                        await deleteTask(t.id);
                        toast.success("Tarefa excluída com sucesso.");
                        await load();
                      } catch (err) {
                        const message =
                          err instanceof Error ? err.message : "Erro ao excluir tarefa.";
                        toast.error(message);
                      }
                    }}
                  >
                    <Trash2 className="size-4" aria-hidden="true" />
                  </Button>
                  <Button asChild variant="ghost" size="sm">
                    <Link to={`/app/tarefas/${t.id}`} onClick={(e) => e.stopPropagation()}>
                      Abrir <Bot className="ml-2 size-4" aria-hidden="true" />
                    </Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
