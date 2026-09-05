/**
 * Simulated authentication (Etapa 1 — MVP).
 * No backend: credentials are checked locally and the session lives in localStorage.
 * Replace with real auth in a future stage.
 */

export const SESSION_STORAGE_KEY = "bsmart-session";

export type Session = {
  re: string;
  name: string;
  role: string;
  signedInAt: string;
};

const DEMO_USER = {
  re: "123456",
  password: "demo123",
  name: "Colaborador Demo",
  role: "Analista",
};

export type LoginResult = { ok: true; session: Session } | { ok: false; message: string };

/** Simulates a network round-trip and validates credentials. */
export async function login(re: string, password: string): Promise<LoginResult> {
  await new Promise((r) => setTimeout(r, 700));
  const cleanRe = re.replace(/\D/g, "");
  if (cleanRe === DEMO_USER.re && password === DEMO_USER.password) {
    const session: Session = {
      re: DEMO_USER.re,
      name: DEMO_USER.name,
      role: DEMO_USER.role,
      signedInAt: new Date().toISOString(),
    };
    return { ok: true, session };
  }
  return { ok: false, message: "RE ou senha inválidos. Verifique os dados e tente novamente." };
}

export function saveSession(session: Session, remember: boolean) {
  const store = remember ? localStorage : sessionStorage;
  store.setItem(SESSION_STORAGE_KEY, JSON.stringify(session));
}

export function getSession(): Session | null {
  if (typeof window === "undefined") return null;
  const raw =
    localStorage.getItem(SESSION_STORAGE_KEY) ?? sessionStorage.getItem(SESSION_STORAGE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Session;
  } catch {
    return null;
  }
}

export function clearSession() {
  localStorage.removeItem(SESSION_STORAGE_KEY);
  sessionStorage.removeItem(SESSION_STORAGE_KEY);
}
