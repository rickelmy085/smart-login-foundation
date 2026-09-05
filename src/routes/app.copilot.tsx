import { createFileRoute } from "@tanstack/react-router";
import { Bot, Paperclip, SendHorizonal, Sparkles } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { suggestedPrompts } from "@/lib/mock-data";

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

  function submit(e?: React.FormEvent) {
    e?.preventDefault();
    if (!value.trim()) return;
    toast.info("O envio de mensagens será habilitado na próxima etapa.", {
      description: "Sua pergunta foi registrada apenas visualmente.",
    });
  }

  return (
    <div className="mx-auto flex min-h-[calc(100svh-8rem)] max-w-3xl flex-col">
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
          placeholder="Pergunte algo ao ABIS…"
          rows={2}
          className="min-h-0 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
        />
        <div className="flex items-center justify-between px-1 pt-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Anexar arquivo"
            onClick={() => toast.info("Anexos disponíveis em uma próxima etapa.")}
          >
            <Paperclip />
          </Button>
          <div className="flex items-center gap-3">
            <span className="hidden text-[11px] text-muted-foreground sm:inline">
              Enter para enviar · Shift+Enter para nova linha
            </span>
            <Button type="submit" size="icon" aria-label="Enviar" disabled={!value.trim()}>
              <SendHorizonal />
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
}
