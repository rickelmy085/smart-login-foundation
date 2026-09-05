import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useId, useState } from "react";
import { AlertCircle, ArrowRight, Loader2, ShieldCheck, Sparkles, Zap } from "lucide-react";
import { toast } from "sonner";

import { BrandLogo } from "@/components/brand-logo";
import { PasswordInput } from "@/components/password-input";
import { ThemeToggle } from "@/components/theme-toggle";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getSession, login, saveSession } from "@/lib/auth";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Entrar — ABIS" },
      {
        name: "description",
        content: "Acesse o ABIS — Agentic Banking Intelligence System com seu RE e senha.",
      },
      { property: "og:title", content: "Entrar — ABIS" },
      {
        property: "og:description",
        content: "Acesse o ABIS — Agentic Banking Intelligence System com seu RE e senha.",
      },
    ],
  }),
  component: LoginPage,
});

const highlights = [
  { icon: Sparkles, title: "Inteligência operacional", text: "Conhecimento corporativo em linguagem natural." },
  { icon: ShieldCheck, title: "Segurança bancária", text: "Acesso restrito a colaboradores autorizados." },
  { icon: Zap, title: "Produtividade", text: "Menos busca, mais decisão no dia a dia." },
];

function LoginPage() {
  const navigate = useNavigate();
  const ids = { re: useId(), pass: useId(), remember: useId(), error: useId() };
  const [re, setRe] = useState("");
  const [password, setPassword] = useState("");
  const [remember, setRemember] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    getSession().then((s) => {
      if (s) navigate({ to: "/app", replace: true });
    });
  }, [navigate]);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    if (!re.trim() || !password) {
      setError("Informe seu RE e sua senha para continuar.");
      return;
    }
    setLoading(true);
    const result = await login(re, password);
    setLoading(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    saveSession(result.session, remember);
    toast.success(`Bem-vindo ao ABIS, ${result.session.name.split(" ")[0]}!`);
    navigate({ to: "/app", replace: true });
  }

  return (
    <div className="grid min-h-svh lg:grid-cols-[1.1fr_1fr]">
      {/* Brand panel */}
      <aside className="relative hidden overflow-hidden bg-gradient-hero text-hero-foreground lg:flex lg:flex-col lg:justify-between lg:p-12">
        <div className="grid-overlay pointer-events-none absolute inset-0" aria-hidden="true" />
        <div
          className="pointer-events-none absolute -right-32 -top-32 size-[28rem] rounded-full bg-brand/25 blur-3xl"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute -bottom-40 -left-20 size-[24rem] rounded-full bg-primary-glow/30 blur-3xl"
          aria-hidden="true"
        />

        <BrandLogo inverted size="md" className="relative" />

        <div className="relative max-w-lg">
          <p className="mb-4 inline-flex items-center gap-2 rounded-full border border-hero-foreground/20 bg-hero-foreground/10 px-3 py-1 text-xs font-semibold uppercase tracking-widest">
            <span className="size-1.5 rounded-full bg-brand" aria-hidden="true" />
            MVP · Etapa 1
          </p>
          <h2 className="text-4xl font-bold leading-tight xl:text-5xl">
            Conhecimento certo. No momento certo.
          </h2>
          <p className="mt-4 text-base text-hero-foreground/75">
            Sistema inteligente para facilitar o acesso ao conhecimento operacional e apoiar
            decisões no ambiente bancário.
          </p>

          <ul className="mt-10 space-y-4">
            {highlights.map(({ icon: Icon, title, text }) => (
              <li key={title} className="flex items-start gap-3">
                <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-hero-foreground/10">
                  <Icon className="size-4 text-brand" aria-hidden="true" />
                </span>
                <div>
                  <p className="font-semibold">{title}</p>
                  <p className="text-sm text-hero-foreground/70">{text}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-hero-foreground/50">
          © {new Date().getFullYear()} ABIS. Uso interno e confidencial.
        </p>
      </aside>

      {/* Form panel */}
      <main className="relative flex flex-col bg-background">
        <header className="flex items-center justify-between p-4 sm:p-6">
          <BrandLogo size="sm" className="lg:invisible" />
          <ThemeToggle />
        </header>

        <div className="flex flex-1 items-center justify-center px-4 pb-12 sm:px-8">
          <div className="w-full max-w-sm">
            <h1 className="text-3xl font-bold text-foreground">Acesse sua conta</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Use seu RE corporativo e a senha cadastrada.
            </p>

            <form onSubmit={onSubmit} noValidate className="mt-8 space-y-5">
              <div className="space-y-2">
                <Label htmlFor={ids.re}>RE (Registro de Empregado)</Label>
                <Input
                  id={ids.re}
                  name="re"
                  inputMode="numeric"
                  autoComplete="username"
                  placeholder="000000"
                  value={re}
                  onChange={(e) => setRe(e.target.value)}
                  aria-invalid={!!error}
                  aria-describedby={error ? ids.error : undefined}
                  className="h-11"
                  required
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor={ids.pass}>Senha</Label>
                  <button
                    type="button"
                    className="text-xs font-medium text-primary underline-offset-4 hover:underline"
                    onClick={() =>
                      toast.info("Recuperação de senha estará disponível em uma próxima etapa.")
                    }
                  >
                    Esqueci minha senha
                  </button>
                </div>
                <PasswordInput
                  id={ids.pass}
                  name="password"
                  autoComplete="current-password"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  aria-invalid={!!error}
                  aria-describedby={error ? ids.error : undefined}
                  className="h-11"
                  required
                />
              </div>

              <div className="flex items-center gap-2">
                <Checkbox
                  id={ids.remember}
                  checked={remember}
                  onCheckedChange={(v) => setRemember(v === true)}
                />
                <Label htmlFor={ids.remember} className="font-normal text-muted-foreground">
                  Manter conectado neste dispositivo
                </Label>
              </div>

              {error && (
                <Alert variant="destructive" id={ids.error} role="alert">
                  <AlertCircle className="size-4" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}

              <Button type="submit" size="lg" className="h-11 w-full" disabled={loading}>
                {loading ? (
                  <>
                    <Loader2 className="animate-spin" aria-hidden="true" />
                    Autenticando…
                  </>
                ) : (
                  <>
                    Entrar
                    <ArrowRight aria-hidden="true" />
                  </>
                )}
              </Button>
            </form>

            <div className="mt-8 rounded-lg border border-dashed border-border bg-muted/50 p-4 text-xs text-muted-foreground">
              <p className="font-semibold text-foreground">Ambiente de demonstração</p>
              <p className="mt-1">
                RE <code className="rounded bg-background px-1 py-0.5 font-mono">123456</code> ·
                senha <code className="rounded bg-background px-1 py-0.5 font-mono">demo123</code>
              </p>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
