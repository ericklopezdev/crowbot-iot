import { createContext, useContext, useState, type ReactNode } from "react";
import { api, USE_MOCKS } from "../api";
import { clearToken, getToken, setToken } from "../api/client";

interface AuthCtx {
  isAuthed: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  // En modo mock siempre autenticado para ir directo al dashboard.
  const [isAuthed, setIsAuthed] = useState<boolean>(USE_MOCKS || !!getToken());

  const login = async (email: string, password: string) => {
    const res = await api.login(email, password);
    setToken(res.access_token);
    setIsAuthed(true);
  };

  const logout = () => {
    clearToken();
    setIsAuthed(false);
  };

  return <Ctx.Provider value={{ isAuthed, login, logout }}>{children}</Ctx.Provider>;
}

export function useAuth(): AuthCtx {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
