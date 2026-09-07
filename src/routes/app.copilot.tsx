import { createFileRoute } from "@tanstack/react-router";
import { Bot, Download, Paperclip, SendHorizonal, Sparkles, User, AlertCircle, FileText, Globe } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { suggestedPrompts } from "@/lib/mock-data";
import { sendQuestion, type ChatMessage, type Source } from "@/lib/chat";
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
      { name: "description", content: "Área inicial do assistente inteligente ABIS." },
      { property: "og:title", content: "ABIS — Agentic Banking Intelligence System" },
      { property: "og:description", content: "Área inicial do assistente inteligente ABIS." },
    ],
  }),
  component: AbisPage,
});

const capabilities = [
  { title: "Consultar políticas", text: "Perguntas sobre normas, benefícios e procedimentos." },
  { title: "Resumir documentos", text: "Sínteses objetivas de manuais e comunicados." },
  { title: "Orientar fluxos", text: "Passo a passo de processos internos." },
];

function AbisPage() {
  const [value, setValue] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [allowWebSearch, setAllowWebSearch] = useState(false);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  async function submit(e?: React.FormEvent) {
    e?.preventDefault();
    const question = value.trim();
    if (!question || loading) return;

    setError(null);
    setMessages((prev) => [...prev, { id: uuid(), role: "user", content: question }]);
    setValue("");
    setLoading(true);

    try {
      const data = await sendQuestion(question, allowWebSearch);

      const isNoEvidence = /normativos dispon.i?veis n.?o trazem informa.?.?o suficiente/i.test(data.answer);

      setMessages((prev) => [
        ...prev,
        {
          id: uuid(),
          role: "assistant",
          content: data.answer,
          sources: data.sources,
          documentRunId: data.documentRunId,
          documentDocxUrl: data.documentDocxUrl,
          documentPdfUrl: data.documentPdfUrl,
          error: isNoEvidence ? "no_evidence" : undefined,
        },
      ]);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Erro ao consultar o assistente.";
      setError(message);
      setMessages((prev) => [
        ...prev,
        {
          id: uuid(),
          role: "assistant",
          content: "Não foi possível obter uma resposta no momento.",
          error: message,
        },
      ]);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100svh-8rem)] max-w-3xl flex-col">
      <div className="flex flex-1 flex-col">
        {messages.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center text-center">
            <span className="grid size-16 place-items-center rounded-2xl bg-gradient-hero text-hero-foreground shadow-elegant">
              <Bot className="size-8" aria-hidden="true" />
            </span>
            <h2 className="mt-6 text-2xl font-bold sm:text-3xl">Olá. Como posso ajudar hoje?</h2>
            <p className="mt-2 max-w-md text-sm text-muted-foreground">
              Consulte conhecimento corporativo, normativos e informações operacionais em linguagem natural.
            </p>

            <ul className="mt-8 grid w-full gap-3 sm:grid-cols-3">
              {capabilities.map((c) => (
                <li key={c.title} className="rounded-xl border border-border bg-card p-4 text-left shadow-panel">
                  <p className="text-sm font-semibold">{c.title}</p>
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
          <div className="flex-1 space-y-4 overflow-y-auto pb-4">
            {messages.map((m) => (
              <MessageBubble key={m.id} message={m} />
            ))}
            {loading && (
              <div className="flex items-start gap-3">
                <span className="mt-1 grid size-8 shrink-0 place-items-center rounded-full bg-muted text-xs">
                  <Bot className="size-4" aria-hidden="true" />
                </span>
                <div className="rounded-2xl border border-border bg-card px-4 py-3 text-sm">
                  <span className="inline-flex items-center gap-1 text-muted-foreground">
                    <span className="size-2 animate-pulse rounded-full bg-foreground/60" />
                    <span className="size-2 animate-pulse rounded-full bg-foreground/60 [animation-delay:0.15s]" />
                    <span className="size-2 animate-pulse rounded-full bg-foreground/60 [animation-delay:0.3s]" />
                  </span>
                </div>
              </div>
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
        {isUser ? <User className="size-4" aria-hidden="true" /> : <Bot className="size-4" aria-hidden="true" />}
      </span>

      <div className={`flex max-w-[85%] flex-col ${isUser ? "items-end" : "items-start"}`}>
        <div
          className={`rounded-2xl px-4 py-3 text-sm leading-relaxed ${
            isUser
              ? "bg-primary text-primary-foreground"
              : "border border-border bg-card"
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
                  p: ({ node, children, ...props }) => (
                    <p className="mb-2 leading-relaxed" {...props}>{children}</p>
                  ),
                  h1: ({ node, children, ...props }) => (
                    <h1 className="my-3 text-xl font-bold" {...props}>{children}</h1>
                  ),
                  h2: ({ node, children, ...props }) => (
                    <h2 className="my-3 text-lg font-bold" {...props}>{children}</h2>
                  ),
                  h3: ({ node, children, ...props }) => (
                    <h3 className="my-2 text-base font-semibold" {...props}>{children}</h3>
                  ),
                  ul: ({ node, children, ...props }) => (
                    <ul className="my-2 ml-4 list-disc" {...props}>{children}</ul>
                  ),
                  ol: ({ node, children, ...props }) => (
                    <ol className="my-2 ml-4 list-decimal" {...props}>{children}</ol>
                  ),
                  li: ({ node, children, ...props }) => (
                    <li className="mb-1" {...props}>{children}</li>
                  ),
                  strong: ({ node, children, ...props }) => (
                    <strong className="font-semibold" {...props}>{children}</strong>
                  ),
                  blockquote: ({ node, children, ...props }) => (
                    <blockquote className="my-2 border-l-2 border-border pl-4 italic text-muted-foreground" {...props}>{children}</blockquote>
                  ),
                  code: ({ node, inline, className, children, ...props }) => {
                    const match = /language-(\w+)/.exec(className || "");
                    const codeClass = match
                      ? `language-${match[1]}`
                      : inline
                        ? "inline-code"
                        : undefined;
                    return (
                      <code
                        className={codeClass}
                        style={inline ? { background: "var(--color-muted)", borderRadius: "0.35rem", padding: "0.1rem 0.3rem" } : undefined}
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
            <div className="mt-2 flex items-center gap-2 text-xs text-red-500">
              <AlertCircle className="size-3.5" aria-hidden="true" />
              <span>{message.error}</span>
            </div>
          )}
        </div>

        {!isUser && message.sources && message.sources.length > 0 && (
          <div className="mt-2 w-full space-y-2">
            <p className="text-xs font-semibold text-muted-foreground">Fontes normativas</p>
            <div className="grid gap-2">
              {message.sources.map((src, idx) => (
                <SourceCard key={`${src.documentId}-${src.chunkOrd}-${idx}`} source={src} />
              ))}
            </div>
          </div>
        )}

        {!isUser && message.error === "no_evidence" && (
          <div className="mt-2 flex items-start gap-2 rounded-xl border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning-foreground">
            <AlertCircle className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
            <span>
              Esta resposta pode não refletir a base normativa atual. Confira o normativo oficial ou contate a área
              responsável.
            </span>
          </div>
        )}

        {!isUser && message.documentRunId && (
          <div className="mt-2 flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              onClick={async () => {
                try {
                  await downloadDocument(
                    `/api/documents/${message.documentRunId}/docx`,
                    `ABIS-documento-${message.documentRunId}.docx`,
                  );
                } catch (err) {
                  toast.error(err instanceof Error ? err.message : "Erro ao baixar documento.");
                }
              }}
            >
              <Download className="mr-2 size-4" aria-hidden={true} />
              Baixar DOCX
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={async () => {
                try {
                  await downloadDocument(
                    `/api/documents/${message.documentRunId}/pdf`,
                    `ABIS-documento-${message.documentRunId}.pdf`,
                  );
                } catch (err) {
                  toast.error(err instanceof Error ? err.message : "Erro ao baixar documento.");
                }
              }}
            >
              <Download className="mr-2 size-4" aria-hidden={true} />
              Baixar PDF
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function SourceCard({ source }: { source: Source }) {
  return (
    <div className="rounded-xl border border-border bg-card/60 p-3">
      <div className="flex items-center gap-2">
        <FileText className="size-3.5 text-muted-foreground" aria-hidden="true" />
        <p className="text-xs font-semibold line-clamp-1">{source.title || source.documentId}</p>
      </div>
      <p className="mt-1 line-clamp-3 text-xs text-muted-foreground">{source.snippet || source.content}</p>
      <p className="mt-1 text-[11px] text-muted-foreground/80">
        score {typeof source.score === "number" ? source.score.toFixed(2) : "—"}
      </p>
    </div>
  );
}
