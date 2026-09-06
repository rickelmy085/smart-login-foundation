import { createFileRoute, Link, useParams } from "@tanstack/react-router";
import { ArrowLeft, Bot, FileText, Loader2, SendHorizonal, Sparkles, Download } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  getTask,
  processTask,
  sendMessage,
  validateTask,
  generateDocument,
  listDocuments,
  getTaskSources,
  type Task,
  type MissingField,
} from "@/lib/workflow";

export const Route = createFileRoute("/app/tarefas/$id")({
  head: ({ params }) => ({
    meta: [
      { title: `Tarefa ${params.id} — ABIS` },
      { name: "description", content: "Detalhamento da tarefa agentic." },
    ],
  }),
  component: TaskDetailPage,
});

const INTENT_LABEL: Record<string, string> = {
  knowledge_query: "Consulta de conhecimento",
  procedure_query: "Consulta de procedimento",
  document_generation: "Geração de documento",
  form_completion: "Preenchimento de formulário",
  approval_check: "Verificação de aprovação",
  requirement_check: "Verificação de requisitos",
  workflow_execution: "Execução de fluxo",
};

function TaskDetailPage() {
  const { id } = useParams({ from: "/app/tarefas/$id" });
  const [task, setTask] = useState<Task | null>(null);
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [message, setMessage] = useState("");
  const [logs, setLogs] = useState<Array<{ text: string; kind: "user" | "assistant" | "system" }>>([]);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const data = await getTask(id);
        if (!cancelled) {
          setTask(data);
          setLogs((prev) => [
            ...prev,
            { text: `Status: ${data.status}`, kind: "system" },
          ]);
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Erro ao carregar tarefa.";
        if (!cancelled) toast.error(msg);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [id]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const refresh = async () => {
    try {
      const data = await getTask(id);
      setTask(data);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro ao atualizar tarefa.";
      toast.error(msg);
    }
  };

  async function startProcessing() {
    setProcessing(true);
    try {
      const res = await processTask(id);
      setLogs((prev) => [
        ...prev,
        { text: `Processamento iniciado. ${res.message ?? ""}`, kind: "system" },
      ]);
      await refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro ao processar tarefa.";
      toast.error(msg);
    } finally {
      setProcessing(false);
    }
  }

  async function submitMessage(e?: React.FormEvent) {
    e?.preventDefault();
    const text = message.trim();
    if (!text) return;
    setLogs((prev) => [...prev, { text, kind: "user" }]);
    setMessage("");
    try {
      const res = await sendMessage(id, text);
      setLogs((prev) => [
        ...prev,
        { text: res.answer, kind: "assistant" },
      ]);
      await refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro ao enviar mensagem.";
      toast.error(msg);
    }
  }

  async function runValidation() {
    setProcessing(true);
    try {
      const res = await validateTask(id);
      setLogs((prev) => [...prev, { text: res.answer, kind: "assistant" }]);
      await refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro ao validar tarefa.";
      toast.error(msg);
    } finally {
      setProcessing(false);
    }
  }

  async function runGeneration() {
    setProcessing(true);
    try {
      const res = await generateDocument(id);
      setLogs((prev) => [
        ...prev,
        { text: `Documento gerado: ${res.documentRunId}`, kind: "system" },
      ]);
      await refresh();
      toast.success("Documento gerado com sucesso.");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro ao gerar documento.";
      toast.error(msg);
    } finally {
      setProcessing(false);
    }
  }

  if (loading) {
    return (
      <div className="mx-auto flex min-h-[calc(100svh-8rem)] items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" aria-hidden="true" />
      </div>
    );
  }

  if (!task) {
    return (
      <div className="mx-auto max-w-3xl">
        <p className="text-sm text-muted-foreground">Tarefa não encontrada.</p>
        <Button asChild variant="ghost" className="mt-4">
          <Link to="/app/tarefas">
            <ArrowLeft className="mr-2 size-4" aria-hidden="true" />
            Voltar
          </Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <div className="flex items-center gap-3">
        <Button asChild variant="ghost" size="icon">
          <Link to="/app/tarefas">
            <ArrowLeft className="size-4" aria-hidden="true" />
          </Link>
        </Button>
        <div className="min-w-0">
          <h2 className="truncate text-xl font-bold">{task.originalRequest}</h2>
          <p className="text-xs text-muted-foreground">
            {INTENT_LABEL[task.intent] ?? task.intent} · {new Date(task.createdAt).toLocaleString("pt-BR")}
          </p>
        </div>
        <div className="ml-auto">
          <Badge variant="outline" className={task.status === "generated" || task.status === "completed" ? "bg-success/15 text-success border-success/30" : task.status === "blocked" ? "bg-destructive/15 text-destructive border-destructive/30" : "bg-warning/20 text-warning-foreground dark:text-warning border-warning/40"}>
            {task.status}
          </Badge>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <Card className="shadow-panel">
          <CardHeader>
            <CardTitle className="text-base">Requisitos</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {task.requirements.length === 0 ? (
              <p className="text-xs text-muted-foreground">Nenhum requisito identificado ainda.</p>
            ) : (
              task.requirements.map((r) => (
                <div key={r.id} className="rounded-lg border border-border p-2">
                  <p className="text-xs font-medium">{r.label}</p>
                  <p className="text-[11px] text-muted-foreground">{r.sourceDocument}</p>
                </div>
              ))
            )}
          </CardContent>
        </Card>

        <Card className="shadow-panel">
          <CardHeader>
            <CardTitle className="text-base">Dados informados</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {Object.keys(task.data).length === 0 ? (
              <p className="text-xs text-muted-foreground">Nenhum dado coletado ainda.</p>
            ) : (
              Object.entries(task.data).map(([k, v]) => (
                <div key={k} className="rounded-lg border border-border p-2">
                  <p className="text-xs font-medium">{k}</p>
                  <p className="text-xs text-muted-foreground">{v}</p>
                </div>
              ))
            )}
          </CardContent>
        </Card>
      </div>

      <Card className="shadow-panel">
        <CardHeader>
          <CardTitle className="text-base">Fluxo</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {task.status === "detected" && (
            <Button onClick={startProcessing} disabled={processing}>
              {processing && <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />}
              Iniciar processamento
            </Button>
          )}
          {(task.status === "collecting_data" || task.status === "validating") && (
            <form onSubmit={submitMessage} className="flex w-full flex-col gap-2 sm:flex-row">
              <Textarea
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Informe os dados solicitados..."
                rows={2}
                className="min-h-0 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
              />
              <Button type="submit" size="icon" disabled={!message.trim() || processing} aria-label="Enviar">
                <SendHorizonal />
              </Button>
            </form>
          )}
          {task.status === "collecting_data" && (
            <Button variant="secondary" onClick={runValidation} disabled={processing}>
              Validar dados
            </Button>
          )}
          {task.status === "ready_to_generate" && (
            <Button onClick={runGeneration} disabled={processing}>
              {processing && <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />}
              Gerar documento
            </Button>
          )}
        </CardContent>
      </Card>

      {task.missingFields && task.missingFields.length > 0 && (
        <Card className="shadow-panel">
          <CardHeader>
            <CardTitle className="text-base">Campos obrigatórios pendentes</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {task.missingFields.map((f) => (
              <Badge key={f.fieldName} variant="outline" className="border-warning/40 text-warning-foreground">
                {f.label || f.fieldName}
              </Badge>
            ))}
          </CardContent>
        </Card>
      )}

      {task.status === "generated" && (
        <Card className="shadow-panel">
          <CardHeader>
            <CardTitle className="text-base">Documento gerado</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            <Button asChild variant="secondary">
              <a href={`/api/documents/${id}/docx`} download>
                <Download className="mr-2 size-4" aria-hidden="true" />
                Baixar DOCX
              </a>
            </Button>
            <Button asChild variant="secondary">
              <a href={`/api/documents/${id}/pdf`} download>
                <Download className="mr-2 size-4" aria-hidden="true" />
                Baixar PDF
              </a>
            </Button>
          </CardContent>
        </Card>
      )}

      <Card className="shadow-panel">
        <CardHeader>
          <CardTitle className="text-base">Registro da tarefa</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {logs.map((log, idx) => (
            <div key={idx} className={`flex items-start gap-3 ${log.kind === "user" ? "flex-row-reverse" : ""}`}>
              <span className={`mt-1 grid size-7 shrink-0 place-items-center rounded-full text-[10px] ${log.kind === "user" ? "bg-primary text-primary-foreground" : log.kind === "assistant" ? "bg-muted" : "bg-accent text-accent-foreground"}`}>
                <Bot className="size-3.5" aria-hidden="true" />
              </span>
              <div className={`max-w-[85%] rounded-2xl px-3 py-2 text-xs leading-relaxed ${log.kind === "user" ? "bg-primary text-primary-foreground" : log.kind === "assistant" ? "border border-border bg-card" : "bg-muted text-muted-foreground"}`}>
                {log.text}
              </div>
            </div>
          ))}
          <div ref={endRef} />
        </CardContent>
      </Card>
    </div>
  );
}
