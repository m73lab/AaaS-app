import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import { api } from '../lib/api';

export interface SessionUser {
  id: string;
  email: string;
  role: string;
}

interface ClientSessionValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const ClientSessionContext = createContext<ClientSessionValue | null>(null);

export function ClientSessionProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(() => {
    const raw = localStorage.getItem('aas_user');
    return raw ? JSON.parse(raw) : null;
  });

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.post<{ token: string; admin: SessionUser }>('/v1/aas/auth/login', {
      email,
      password,
    });
    localStorage.setItem('aas_token', res.token);
    localStorage.setItem('aas_user', JSON.stringify(res.admin));
    setUser(res.admin);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('aas_token');
    localStorage.removeItem('aas_user');
    setUser(null);
  }, []);

  const value = useMemo<ClientSessionValue>(
    () => ({ user, isAuthenticated: !!user, login, logout }),
    [user, login, logout],
  );

  return <ClientSessionContext.Provider value={value}>{children}</ClientSessionContext.Provider>;
}

export function useClientSession(): ClientSessionValue {
  const ctx = useContext(ClientSessionContext);
  if (!ctx) throw new Error('useClientSession must be used within ClientSessionProvider');
  return ctx;
}
