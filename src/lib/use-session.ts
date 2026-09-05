import { useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { clearSession, getSession, type Session } from "@/lib/auth";

/**
 * Client-side session hook for the simulated auth.
 * Redirects to the login page when no session exists.
 */
export function useSession() {
  const navigate = useNavigate();
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const s = getSession();
    if (!s) {
      navigate({ to: "/", replace: true });
      return;
    }
    setSession(s);
    setReady(true);
  }, [navigate]);

  const signOut = useCallback(() => {
    clearSession();
    navigate({ to: "/", replace: true });
  }, [navigate]);

  return { session, ready, signOut };
}
