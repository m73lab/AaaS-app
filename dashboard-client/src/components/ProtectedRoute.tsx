import type { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';
import { useClientSession } from '../context/ClientSessionContext';

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useClientSession();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}
