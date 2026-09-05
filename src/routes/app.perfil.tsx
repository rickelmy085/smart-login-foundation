import { createFileRoute } from "@tanstack/react-router";
import { Monitor, Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { initials } from "@/components/app-sidebar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { getSession, type Session } from "@/lib/auth";
import { useTheme, type Theme } from "@/lib/theme";
import { cn } from "@/lib/utils";

export const Route = createFileRoute("/app/perfil")({
  head: () => ({
    meta: [
      { title: "Meu perfil — B-Smart Copilot" },
      { name: "description", content: "Dados do colaborador e preferências do B-Smart Copilot." },
      { property: "og:title", content: "Meu perfil — B-Smart Copilot" },
      { property: "og:description", content: "Dados do colaborador e preferências do B-Smart Copilot." },
    ],
  }),
  component: ProfilePage,
});

const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: "light", label: "Claro", icon: Sun },
  { value: "dark", label: "Escuro", icon: Moon },
  { value: "system", label: "Sistema", icon: Monitor },
];

const dateFmt = new Intl.DateTimeFormat("pt-BR", { dateStyle: "long", timeStyle: "short" });

function ProfilePage() {
  const [session, setSession] = useState<Session | null>(null);
  const { theme, setTheme } = useTheme();
  const [notify, setNotify] = useState(true);
  const [digest, setDigest] = useState(false);

  useEffect(() => setSession(getSession()), []);
  if (!session) return null;

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <Card className="overflow-hidden shadow-panel">
        <div className="h-24 bg-gradient-hero" aria-hidden="true" />
        <CardContent className="-mt-10 flex flex-col gap-4 sm:flex-row sm:items-end">
          <span className="grid size-20 shrink-0 place-items-center rounded-2xl border-4 border-card bg-primary font-display text-2xl font-bold text-primary-foreground shadow-panel">
            {initials(session.name)}
          </span>
          <div className="flex-1 pb-1">
            <h2 className="text-xl font-bold">{session.name}</h2>
            <p className="text-sm text-muted-foreground">
              {session.role} · RE {session.re}
            </p>
          </div>
          <Badge className="mb-1 w-fit bg-success/15 text-success hover:bg-success/15">Ativo</Badge>
        </CardContent>
      </Card>

      <Card className="shadow-panel">
        <CardHeader>
          <CardTitle className="text-base">Dados do colaborador</CardTitle>
          <CardDescription>Informações sincronizadas do cadastro corporativo (somente leitura).</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <Field label="Nome completo" value={session.name} />
          <Field label="RE" value={session.re} />
          <Field label="Cargo" value={session.role} />
          <Field label="E-mail corporativo" value={`re${session.re}@bsmart.com.br`} />
          <Field label="Último acesso" value={dateFmt.format(new Date(session.signedInAt))} />
          <Field label="Perfil de acesso" value="Colaborador" />
        </CardContent>
      </Card>

      <Card className="shadow-panel">
        <CardHeader>
          <CardTitle className="text-base">Preferências</CardTitle>
          <CardDescription>Personalize sua experiência no Copilot.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <fieldset>
            <legend className="mb-3 text-sm font-medium">Aparência</legend>
            <div className="grid grid-cols-3 gap-3" role="radiogroup" aria-label="Tema">
              {themeOptions.map(({ value, label, icon: Icon }) => {
                const active = theme === value;
                return (
                  <button
                    key={value}
                    type="button"
                    role="radio"
                    aria-checked={active}
                    onClick={() => setTheme(value)}
                    className={cn(
                      "flex flex-col items-center gap-2 rounded-xl border p-4 text-sm transition-colors",
                      active
                        ? "border-primary bg-accent text-accent-foreground"
                        : "border-border hover:bg-accent/50",
                    )}
                  >
                    <Icon className="size-5" aria-hidden="true" />
                    {label}
                  </button>
                );
              })}
            </div>
          </fieldset>

          <Separator />

          <div className="space-y-4">
            <ToggleRow
              id="notify"
              label="Notificações no app"
              description="Avisos sobre respostas concluídas e novidades."
              checked={notify}
              onCheckedChange={setNotify}
            />
            <ToggleRow
              id="digest"
              label="Resumo semanal por e-mail"
              description="Receba um resumo das suas consultas toda segunda-feira."
              checked={digest}
              onCheckedChange={setDigest}
            />
          </div>

          <div className="flex justify-end">
            <Button onClick={() => toast.success("Preferências salvas (simulação).")}>
              Salvar preferências
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  const id = `field-${label.replace(/\W+/g, "-").toLowerCase()}`;
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-muted-foreground">
        {label}
      </Label>
      <Input id={id} value={value} readOnly className="bg-muted/40" />
    </div>
  );
}

function ToggleRow({
  id,
  label,
  description,
  checked,
  onCheckedChange,
}: {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div>
        <Label htmlFor={id}>{label}</Label>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}
