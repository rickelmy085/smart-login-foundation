/**
 * Real authentication backed by the ABIS API.
 */

import { apiFetch, clearToken, setToken } from "@/lib/api";

export const SESSION_STORAGE_KEY = "bsmart-session";

export type Session = {
  re: string;
  name: string;
  role: string;
  signedInAt: string;
};

export type LoginResult = { ok: true; session: Session } | { ok: false; message: string };

export async function login(re: string, password: string): Promise<LoginResult> {
  try {
    const res = await apiFetch("/api/login", {
      method: "POST",
      body: JSON.stringify({ re, password, remember: true }),
    });

    const data = (await res.json()) as {
      token: string;
      employee: { id: string; re: string; name: string; role: string; email: string };
    };

    const token = data.token;
    if (!token) {
      return { ok: false, message: "RE ou senha inválidos. Verifique os dados e tente novamente." };
    }

    setToken(token);
    const session: Session = {
      re: data.employee.re,
      name: data.employee.name,
      role: data.employee.role,
      signedInAt: new Date().toISOString(),
    };
    saveSession(session, true);
    return { ok: true, session };
  } catch (error) {
    const message = error instanceof Error ? error.message : "RE ou senha inválidos. Verifique os dados e tente novamente.";
    return { ok: false, message };
  }
}

export function saveSession(session: Session, remember: boolean) {
  const store = remember ? localStorage : sessionStorage;
  store.setItem(SESSION_STORAGE_KEY, JSON.stringify(session));
}

export async function getSession(): Promise<Session | null> {
  const token = localStorage.getItem("bsmart-token");
  if (!token) return null;

  try {
    const res = await apiFetch("/api/me", {
      headers: { Authorization: `Bearer ${token}` },
    });

    const employee = (await res.json()) as { re: string; name: string; role: string };
    const session: Session = {
      re: employee.re,
      name: employee.name,
      role: employee.role,
      signedInAt: new Date().toISOString(),
    };
    saveSession(session, true);
    return session;
  } catch {
    clearSession();
    return null;
  }
}

export function clearSession() {
  localStorage.removeItem(SESSION_STORAGE_KEY);
  localStorage.removeItem("bsmart-token");
}
