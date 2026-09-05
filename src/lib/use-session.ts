import { useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { clearSession, getSession, type Session } from "@/lib/auth";

export function useSession() {
  const navigate = useNavigate();
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;

    getSession().then((s) => {
      if (cancelled) return;
      if (!s) {
        navigate({ to: "/", replace: true });
        return;
      }
      setSession(s);
      setReady(true);
    });

    return () => {
      cancelled = true;
    };
  }, [navigate]);

  const signOut = useCallback(() => {
    clearSession();
    navigate({ to: "/", replace: true });
  }, [navigate]);

  return { session, ready, signOut };
}
