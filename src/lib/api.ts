const API_BASE = (import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8081";

export async function apiFetch(path: string, init: RequestInit = {}) {
  const token = getToken();
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    credentials: "omit",
  });

  if (res.status === 401) {
    clearToken();
    throw new Error("unauthorized");
  }

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `request failed with status ${res.status}`);
  }

  return res;
}

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("bsmart-token");
}

export function setToken(token: string) {
  localStorage.setItem("bsmart-token", token);
}

export function clearToken() {
  localStorage.removeItem("bsmart-token");
}
