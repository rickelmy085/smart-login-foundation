import { createFileRoute } from "@tanstack/react-router";
import {
  Bot,
  Download,
  Paperclip,
  SendHorizonal,
  Sparkles,
  User,
  AlertCircle,
  FileText,
  Globe,
  Loader2,
  Check,
  Clock,
  ArrowRight,
  PauseCircle,
  AlertTriangle,
  FileCheck2,
  BookOpen,
  ShieldCheck,
  XCircle,
  Circle,
  CheckCircle2,
  CircleDot,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { suggestedPrompts } from "@/lib/mock-data";
import {
  type ChatMessage,
  type Source,
  type AgentStatus,
  type AgentPlan,
  type AgentResponse,
  agentProcess,
  agentHumanInput,
  getAgentStatus,
} from "@/lib/chat";
import { downloadDocument } from "@/lib/api";

function uuid() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export const Route = createFileRoute("/app/copilot")({
  head: () => ({
    meta: [
      { title: "ABIS — Agentic Banking Intelligence System" },
      { name: "description", content: "Assistente inteligente ABIS." },
      { property: "og:title", content: "ABIS — Agentic Banking Intelligence System" },
      { property: "og:description", content: "Assistente inteligente ABIS." },
    ],
  }),
  component: AbisPage,
});

const capabilities = [
  {
    title: "Gerar documentos",
    text: "Relatórios operacionais, solicitações de aquisição e outros documentos normatizados.",
    icon: FileCheck2,
  },
  {
    title: "Consultar normativos",
    text: "Perguntas sobre políticas, procedimentos e regras internas da organização.",
    icon: BookOpen,
  },
  {
    title: "Executar workflows",
    text: "Condução guiada de tarefas com validação de requisitos e regras.",
    icon: ShieldCheck,
  },
];

function AbisPage() {
  const [value, setValue] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [allowWebSearch, setAllowWebSearch] = useState(false);
  const [agentState, setAgentState] = useState<AgentStatus>("idle");
  const [currentPlan, setCurrentPlan] = useState<AgentPlan | null>(null);
  const [currentPlanId, setCurrentPlanId] = useState<string | null>(null);
  const [currentGoal, setCurrentGoal] = useState<string>("");
  const [agentResults, setAgentResults] = useState<any[]>([]);
  const [pendingHumanQuestion, setPendingHumanQuestion] = useState<string | null>(null);
  const [humanInputOpen, setHumanInputOpen] = useState(false);
  const [humanInputValue, setHumanInputValue] = useState("");
  const [humanInputLoading, setHumanInputLoading] = useState(false);
  const endRef = useRef<HTMLDivElement>(null);
  const finalizedPlanRef = useRef<string | null>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  // Extrai o texto da resposta final a partir da mensagem ou dos resultados das tools.
  function extractAnswer(data: AgentResponse): string {
    if (data.message && data.message.trim()) return data.message;
    const results = data.results ?? [];
    for (let i = results.length - 1; i >= 0; i--) {
      const out = results[i]?.output;
      if (out && typeof out === "object") {
        if (typeof out.answer === "string" && out.answer.trim()) return out.answer;
        if (typeof out.message === "string" && out.message.trim()) return out.message;
      }
    }
    return "Tarefa concluída.";
  }

  function appendAssistantMessage(goal: string, data: AgentResponse, status: AgentStatus) {
    const isNoEvidence = /normativos dispon.i?veis n.?o trazem informa.?.?o suficiente/i.test(
      data.message ?? "",
    );
    setMessages((prev) => [
      ...prev,
      {
        id: uuid(),
        role: "assistant",
        content: status === "failed" ? data.message || "Falha ao executar a tarefa." : extractAnswer(data),
        sources: data.results?.flatMap((r: any) => r.output?.sources ?? []) ?? [],
        documentRunId: data.results?.find((r: any) => r.tool === "generate_document")?.output
          ?.document_run_id,
        documentDocxUrl: data.results?.find((r: any) => r.tool === "generate_document")?.output
          ?.docx_url,
        documentPdfUrl: data.results?.find((r: any) => r.tool === "generate_document")?.output
          ?.pdf_url,
        error:
          status === "failed"
            ? data.error || "Falha ao executar a tarefa."
            : isNoEvidence
              ? "no_evidence"
              : undefined,
        agentTrace: buildTrace(goal, data, status),
      },
    ]);
  }

  async function submit(e?: React.FormEvent) {
    e?.preventDefault();
    const question = value.trim();
    if (!question || loading) return;

    setError(null);
    setMessages((prev) => [...prev, { id: uuid(), role: "user", content: question }]);
    setValue("");
    setLoading(true);
    setAgentState("planning");
    setAgentResults([]);
    setCurrentPlan(null);
    setCurrentGoal(question);
    finalizedPlanRef.current = null;

    let handedOffToPolling = false;

    try {
      const data = await agentProcess({ goal: question, allowWebSearch });

      if (data.plan) setCurrentPlan(data.plan);
      if (data.plan_id) setCurrentPlanId(data.plan_id);
      if (data.results) setAgentResults(data.results);

      // Erro de planejamento/execução: backend retorna success=false sem status.
      if (!data.success && !data.status) {
        const message = data.error || "Não foi possível processar sua solicitação.";
        setMessages((prev) => [
          ...prev,
          { id: uuid(), role: "assistant", content: message, error: message },
        ]);
        setError(message);
        toast.error(message);
        return;
      }

      if (data.status === "awaiting_human" && data.human_question) {
        setPendingHumanQuestion(data.human_question);
        appendAssistantMessage(data, "awaiting_human");
        setHumanInputOpen(true);
        setAgentState("awaiting_human");
      } else if (data.status === "completed" || data.status === "failed") {
        appendAssistantMessage(data, data.status);
        setAgentState(data.status);
      } else {
        // Execução assíncrona ainda em andamento ("running"): entrega ao polling.
        handedOffToPolling = true;
        setAgentState("executing");
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Erro ao consultar o assistente.";
      setError(message);
      setMessages((prev) => [
        ...prev,
        {
          id: uuid(),
          role: "assistant",
          content: "Não foi possível processar sua solicitação no momento.",
          error: message,
        },
      ]);
      toast.error(message);
    } finally {
      if (!handedOffToPolling) {
        setLoading(false);
        setAgentState((s) => (s === "awaiting_human" || s === "completed" || s === "failed" ? s : "idle"));
      }
    }
  }

  async function handleHumanSubmit() {
    if (!currentPlanId || !humanInputValue.trim()) return;

    setHumanInputLoading(true);
    try {
      const data = await agentHumanInput(currentPlanId, humanInputValue.trim());

      if (data.plan) setCurrentPlan(data.plan);
      if (data.results) setAgentResults(data.results);

      setHumanInputOpen(false);
      setHumanInputValue("");
      setPendingHumanQuestion(null);
      setLoading(true);

      if (data.status === "completed" || data.status === "failed") {
        appendAssistantMessage(data, data.status);
        setAgentState(data.status);
      } else if (data.status === "awaiting_human" && data.human_question) {
        setPendingHumanQuestion(data.human_question);
        appendAssistantMessage(data, "awaiting_human");
        setHumanInputOpen(true);
        setAgentState("awaiting_human");
      } else if (!data.success) {
        const message = data.error || "Não foi possível processar a resposta.";
        setMessages((prev) => [
          ...prev,
          { id: uuid(), role: "assistant", content: message, error: message },
        ]);
        toast.error(message);
        setAgentState("failed");
      } else {
        // Ainda em execução ("running"): continua acompanhando via polling.
        handedOffToPolling = true;
        setAgentState("executing");
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Erro ao processar resposta.";
      toast.error(message);
    } finally {
      setHumanInputLoading(false);
      if (!handedOffToPolling) {
        setLoading(false);
      }
    }
  }

  useEffect(() => {
    if (!currentPlanId || agentState !== "executing") return;
    const planId = currentPlanId;
    const interval = setInterval(async () => {
      try {
        const data = await getAgentStatus(planId);
        if (data.plan) setCurrentPlan(data.plan);
        if (data.results) setAgentResults(data.results);
        if (data.status === "awaiting_human" && data.human_question) {
          if (finalizedPlanRef.current !== planId) {
            finalizedPlanRef.current = planId;
            appendAssistantMessage(data, "awaiting_human");
          }
          setPendingHumanQuestion(data.human_question);
          setHumanInputOpen(true);
          setAgentState("awaiting_human");
          setLoading(false);
        } else if (data.status === "completed" || data.status === "failed") {
          // Adiciona a resposta final do assistente ao chat (uma única vez por plano).
          if (finalizedPlanRef.current !== planId) {
            finalizedPlanRef.current = planId;
            appendAssistantMessage(data, data.status);
            if (data.status === "failed") {
              toast.error(data.error || "Falha ao executar a tarefa.");
            }
          }
          setAgentState(data.status);
          setLoading(false);
        }
      } catch {
        // Falha transitória de rede no polling: tenta novamente no próximo ciclo.
      }
    }, 2500);
    return () => clearInterval(interval);
  }, [currentPlanId, agentState]);

  return (
    <div className="mx-auto flex min-h-[calc(100svh-8rem)] max-w-3xl flex-col">
      <div className="flex flex-1 flex-col">
        {messages.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center text-center px-4">
            <span className="grid size-16 place-items-center rounded-2xl bg-gradient-hero text-hero-foreground shadow-elegant">
              <Bot className="size-8" aria-hidden="true" />
            </span>
            <h2 className="mt-6 text-2xl font-bold sm:text-3xl">Olá. Como posso ajudar hoje?</h2>
            <p className="mt-2 max-w-md text-sm text-muted-foreground">
              Eu entendo o que você precisa fazer e conduzo a tarefa até o resultado, respeitando os
              normativos da organização.
            </p>

            <ul className="mt-8 grid w-full gap-3 sm:grid-cols-3">
              {capabilities.map((c) => (
                <li
                  key={c.title}
                  className="rounded-xl border border-border bg-card p-4 text-left shadow-panel transition-colors hover:border-primary/40"
                >
                  <c.icon className="size-5 text-brand" aria-hidden="true" />
                  <p className="mt-2 text-sm font-semibold">{c.title}</p>
                  <p className="mt-1 text-xs text-muted-foreground">{c.text}</p>
                </li>
              ))}
            </ul>

            <div className="mt-8 w-full">
              <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Experimente perguntar
              </p>
              <div className="flex flex-wrap justify-center gap-2">
                {suggestedPrompts.map((p) => (
                  <button
                    key={p}
                    type="button"
                    onClick={() => setValue(p)}
                    className="rounded-full border border-border bg-card px-3 py-1.5 text-xs transition-colors hover:border-primary/40 hover:bg-accent/50"
                  >
                    <Sparkles className="mr-1 inline size-3 text-brand" aria-hidden="true" />
                    {p}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex-1 space-y-5 overflow-y-auto pb-4">
            {messages.map((m) => (
              <MessageBubble key={m.id} message={m} />
            ))}
            {loading && (
              <AgentLoadingIndicator
                state={agentState}
                plan={currentPlan}
                results={agentResults}
                goal={currentGoal}
              />
            )}
            <div ref={endRef} />
          </div>
        )}
      </div>

      <form
        onSubmit={submit}
        className="sticky bottom-4 mt-8 rounded-2xl border border-border bg-card p-2 shadow-panel"
      >
        <label htmlFor="copilot-input" className="sr-only">
          Sua pergunta
        </label>
        <Textarea
          id="copilot-input"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
          placeholder={messages.length === 0 ? "Pergunte algo ao ABIS…" : "Continuar conversa…"}
          rows={2}
          className="min-h-0 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
          disabled={loading}
        />
        <div className="flex items-center justify-between px-1 pt-1">
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Anexar arquivo"
              disabled
              title="Upload de arquivos em desenvolvimento"
            >
              <Paperclip />
            </Button>
            <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Checkbox
                checked={allowWebSearch}
                onCheckedChange={(checked) => setAllowWebSearch(checked === true)}
              />
              <Globe className="size-3.5" aria-hidden="true" />
              Buscar na internet se não houver normativo
            </label>
          </div>
          <div className="flex items-center gap-3">
            <span className="hidden text-[11px] text-muted-foreground sm:inline">
              Enter para enviar · Shift+Enter para nova linha
            </span>
            <Button type="submit" size="icon" aria-label="Enviar" disabled={!value.trim() || loading}>
              <SendHorizonal />
            </Button>
          </div>
        </div>
      </form>

      <Dialog
        open={humanInputOpen}
        onOpenChange={(open) => {
          if (!open && agentState === "awaiting_human") return;
          setHumanInputOpen(open);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <PauseCircle className="size-5 text-warning" aria-hidden="true" />
              Ação necessária
            </DialogTitle>
            <DialogDescription>
              O ABIS precisa da sua intervenção para continuar.
            </DialogDescription>
          </DialogHeader>
          {pendingHumanQuestion && (
            <div className="rounded-lg border border-border bg-muted/40 p-3 text-sm">
              {pendingHumanQuestion}
            </div>
          )}
          <Textarea
            value={humanInputValue}
            onChange={(e) => setHumanInputValue(e.target.value)}
            placeholder="Informe os dados solicitados…"
            rows={3}
            className="resize-none"
            disabled={humanInputLoading}
          />
          <DialogFooter>
            <Button
              onClick={handleHumanSubmit}
              disabled={!humanInputValue.trim() || humanInputLoading}
            >
              {humanInputLoading && <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />}
              Continuar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function buildTrace(goal: string, data: any, status: AgentStatus) {
  return {
    plan_id: data.plan_id || "",
    goal,
    current_step: data.current_step || 0,
    steps:
      data.plan?.steps?.map((s: any) => ({
        step_key: s.tool,
        name: s.description,
        status: (status === "completed" ? "completed" : s.status ?? "pending") as any,
        order: s.step_number,
        description: s.description,
      })) ?? [],
    next_action: data.next_action,
    status,
    requires_human: status === "awaiting_human",
    human_question: data.human_question,
    created_at: new Date().toISOString(),
  };
}

function AgentLoadingIndicator({
  state,
  plan,
  results,
  goal,
}: {
  state: AgentStatus;
  plan: any;
  results: any[];
  goal: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <span className="mt-1 grid size-8 shrink-0 place-items-center rounded-full bg-muted text-xs">
        <Bot className="size-4 text-primary" aria-hidden="true" />
      </span>
      <div className="flex-1 rounded-2xl border border-border bg-card px-4 py-3 text-sm shadow-panel">
        <div className="flex items-center gap-2">
          <Loader2 className="size-4 animate-spin text-primary" aria-hidden="true" />
          <span className="font-medium">Executando tarefa</span>
        </div>
        {goal && (
          <p className="mt-1 text-xs text-muted-foreground truncate">Objetivo: {goal}</p>
        )}
        {plan?.steps && plan.steps.length > 0 && (
          <div className="mt-3 space-y-1.5">
            {plan.steps.map((step: any, idx: number) => (
              <StepStatusRow key={step.step_number || idx} step={step} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function StepStatusRow({ step }: { step: any }) {
  const status = step.status || "pending";
  if (status === "completed") {
    return (
      <div className="flex items-center gap-2 text-xs">
        <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />
        <span className="text-foreground/90">{step.description}</span>
      </div>
    );
  }
  if (status === "running") {
    return (
      <div className="flex items-center gap-2 text-xs">
        <Loader2 className="size-4 shrink-0 animate-spin text-primary" aria-hidden="true" />
        <span className="font-medium text-primary">{step.description}</span>
      </div>
    );
  }
  if (status === "failed") {
    return (
      <div className="flex items-center gap-2 text-xs">
        <XCircle className="size-4 shrink-0 text-destructive" aria-hidden="true" />
        <span className="text-destructive">{step.description}</span>
      </div>
    );
  }
  return (
    <div className="flex items-center gap-2 text-xs">
      <Circle className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
      <span className="text-muted-foreground">{step.description}</span>
    </div>
  );
}

function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user";

  return (
    <div className={`flex items-start gap-3 ${isUser ? "flex-row-reverse" : ""}`}>
      <span
        className={`mt-1 grid size-8 shrink-0 place-items-center rounded-full text-xs ${
          isUser ? "bg-primary text-primary-foreground" : "bg-muted"
        }`}
      >
        {isUser ? (
          <User className="size-4" aria-hidden="true" />
        ) : (
          <Bot className="size-4 text-primary" aria-hidden="true" />
        )}
      </span>

      <div className={`flex max-w-[85%] flex-col ${isUser ? "items-end" : "items-start"}`}>
        <div
          className={`rounded-2xl px-4 py-3 text-sm leading-relaxed ${
            isUser ? "bg-primary text-primary-foreground" : "border border-border bg-card shadow-panel"
          }`}
        >
          {isUser ? (
            <span className="whitespace-pre-wrap break-words">{message.content}</span>
          ) : (
            <div className="markdown-body max-w-none">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight]}
                components={{
                  p: ({ children, ...props }) => (
                    <p className="mb-2 leading-relaxed last:mb-0" {...props}>
                      {children}
                    </p>
                  ),
                  h1: ({ children, ...props }) => (
                    <h1 className="my-3 text-xl font-bold" {...props}>
                      {children}
                    </h1>
                  ),
                  h2: ({ children, ...props }) => (
                    <h2 className="my-3 text-lg font-bold" {...props}>
                      {children}
                    </h2>
                  ),
                  h3: ({ children, ...props }) => (
                    <h3 className="my-2 text-base font-semibold" {...props}>
                      {children}
                    </h3>
                  ),
                  ul: ({ children, ...props }) => (
                    <ul className="my-2 ml-4 list-disc space-y-1" {...props}>
                      {children}
                    </ul>
                  ),
                  ol: ({ children, ...props }) => (
                    <ol className="my-2 ml-4 list-decimal space-y-1" {...props}>
                      {children}
                    </ol>
                  ),
                  li: ({ children, ...props }) => (
                    <li className="" {...props}>
                      {children}
                    </li>
                  ),
                  strong: ({ children, ...props }) => (
                    <strong className="font-semibold" {...props}>
                      {children}
                    </strong>
                  ),
                  blockquote: ({ children, ...props }) => (
                    <blockquote
                      className="my-2 border-l-2 border-border pl-4 italic text-muted-foreground"
                      {...props}
                    >
                      {children}
                    </blockquote>
                  ),
                  code: ({ className, children, ...props }) => {
                    const match = /language-(\w+)/.exec(className || "");
                    const inline = !match && !className;
                    return (
                      <code
                        className={match ? `language-${match[1]}` : className}
                        style={
                          inline
                            ? {
                                background: "var(--color-muted)",
                                borderRadius: "0.35rem",
                                padding: "0.1rem 0.3rem",
                              }
                            : undefined
                        }
                        {...props}
                      >
                        {children}
                      </code>
                    );
                  },
                }}
              >
                {message.content}
              </ReactMarkdown>
            </div>
          )}
          {message.error && message.error !== "no_evidence" && (
            <div className="mt-2 flex items-center gap-2 text-xs text-destructive">
              <AlertCircle className="size-3.5" aria-hidden="true" />
              <span>{message.error}</span>
            </div>
          )}
        </div>

        {!isUser && message.documentRunId && (
          <DocumentResultCard
            documentRunId={message.documentRunId}
            docxUrl={message.documentDocxUrl}
            pdfUrl={message.documentPdfUrl}
          />
        )}

        {!isUser && message.sources && message.sources.length > 0 && (
          <SourcesList sources={message.sources} />
        )}

        {!isUser && message.error === "no_evidence" && (
          <div className="mt-2 flex items-start gap-2 rounded-xl border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning-foreground">
            <AlertCircle className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
            <span>
              Esta resposta pode não refletir a base normativa atual. Confira o normativo oficial ou
              contate a área responsável.
            </span>
          </div>
        )}

        {!isUser && message.agentTrace && (
          <AgentTraceCard trace={message.agentTrace} />
        )}
      </div>
    </div>
  );
}

function DocumentResultCard({
  documentRunId,
  docxUrl,
  pdfUrl,
}: {
  documentRunId: string;
  docxUrl?: string;
  pdfUrl?: string;
}) {
  const [downloading, setDownloading] = useState<string | null>(null);

  async function handleDownload(format: "docx" | "pdf") {
    setDownloading(format);
    try {
      await downloadDocument(
        `/api/documents/${documentRunId}/${format}`,
        `ABIS-documento-${documentRunId}.${format}`,
      );
      toast.success(`Download do ${format.toUpperCase()} iniciado.`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Erro ao baixar documento.");
    } finally {
      setDownloading(null);
    }
  }

  return (
    <div className="mt-3 w-full overflow-hidden rounded-xl border border-success/40 bg-success/5 shadow-panel">
      <div className="flex items-start gap-3 p-4">
        <span className="grid size-10 shrink-0 place-items-center rounded-lg bg-success/15 text-success">
          <FileCheck2 className="size-5" aria-hidden="true" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-semibold">Documento gerado</p>
            <Badge
              variant="outline"
              className="border-success/40 bg-success/10 text-success"
            >
              Concluído
            </Badge>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Documento disponível para download.
          </p>
        </div>
      </div>
      <div className="flex flex-wrap gap-2 border-t border-border/60 bg-card/40 px-4 py-3">
        <Button
          size="sm"
          onClick={() => handleDownload("docx")}
          disabled={downloading !== null}
        >
          {downloading === "docx" ? (
            <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />
          ) : (
            <Download className="mr-2 size-4" aria-hidden="true" />
          )}
          Baixar DOCX
        </Button>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => handleDownload("pdf")}
          disabled={downloading !== null}
        >
          {downloading === "pdf" ? (
            <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />
          ) : (
            <Download className="mr-2 size-4" aria-hidden="true" />
          )}
          Baixar PDF
        </Button>
      </div>
    </div>
  );
}

function SourcesList({ sources }: { sources: Source[] }) {
  const deduped = sources.reduce<Source[]>((acc, s) => {
    const key = s.title || s.documentId;
    if (!acc.some((a) => (a.title || a.documentId) === key)) acc.push(s);
    return acc;
  }, []);

  return (
    <div className="mt-3 w-full">
      <p className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-muted-foreground">
        <BookOpen className="size-3.5" aria-hidden="true" />
        Fontes utilizadas
      </p>
      <ul className="space-y-1.5">
        {deduped.map((src, idx) => (
          <li
            key={`${src.documentId}-${idx}`}
            className="flex items-start gap-2 rounded-lg border border-border bg-card/60 px-3 py-2 text-xs"
          >
            <FileText className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
            <div className="min-w-0 flex-1">
              <p className="font-medium truncate">{src.title || src.documentId}</p>
              {src.snippet && (
                <p className="mt-0.5 line-clamp-2 text-muted-foreground">{src.snippet}</p>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

function AgentTraceCard({ trace }: { trace: any }) {
  const [expanded, setExpanded] = useState(false);
  if (!trace?.steps?.length) return null;

  const completed = trace.steps.filter((s: any) => s.status === "completed").length;
  const total = trace.steps.length;
  const isCompleted = trace.status === "completed";
  const isFailed = trace.status === "failed";
  const isAwaiting = trace.status === "awaiting_human";

  return (
    <div className="mt-3 w-full overflow-hidden rounded-xl border border-border bg-card/60">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-accent/40"
      >
        <div className="flex items-center gap-2 min-w-0">
          {isCompleted && <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />}
          {isFailed && <XCircle className="size-4 shrink-0 text-destructive" aria-hidden="true" />}
          {isAwaiting && <PauseCircle className="size-4 shrink-0 text-warning" aria-hidden="true" />}
          {!isCompleted && !isFailed && !isAwaiting && (
            <CircleDot className="size-4 shrink-0 text-primary" aria-hidden="true" />
          )}
          <span className="text-xs font-medium truncate">
            {trace.goal || "Execução da tarefa"}
          </span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="text-[11px] text-muted-foreground">
            {completed}/{total}
          </span>
          <Badge
            variant="outline"
            className={
              isCompleted
                ? "border-success/40 bg-success/10 text-success"
                : isFailed
                  ? "border-destructive/40 bg-destructive/10 text-destructive"
                  : isAwaiting
                    ? "border-warning/40 bg-warning/10 text-warning-foreground"
                    : "border-primary/40 bg-primary/10 text-primary"
            }
          >
            {isCompleted
              ? "Concluído"
              : isFailed
                ? "Falhou"
                : isAwaiting
                  ? "Aguardando"
                  : "Em execução"}
          </Badge>
        </div>
      </button>
      {expanded && (
        <div className="border-t border-border px-4 py-3 space-y-1.5">
          {trace.steps.map((step: any, idx: number) => (
            <StepStatusRow key={idx} step={step} />
          ))}
        </div>
      )}
    </div>
  );
}
