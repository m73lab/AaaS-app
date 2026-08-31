import { useQuery } from '@tanstack/react-query';
import { api } from '../../lib/api';
import { queryKeys } from './queryKeys';

export interface Overview {
  totalRequests: number;
  totalPromptTokens: number;
  totalCompletionTokens: number;
  totalEntitiesDetected: number;
  activeTenants: number;
  requestsBlocked: number;
  avgLatencyMs: number;
  topCategories: { category: string; count: number }[];
}

export interface UsagePoint {
  hour: string;
  requests: number;
  tokens: number;
  entities: number;
}

export interface Tenant {
  id: string;
  name: string;
  email: string;
  plan: string;
  rateLimitPerMin: number;
  rateLimitPerHour: number;
  active: boolean;
}

export interface UsageLog {
  id: number;
  tenantId: string;
  timestamp: string;
  sessionId: string;
  model: string | null;
  provider: string | null;
  format: string | null;
  promptTokens: number | null;
  completionTokens: number | null;
  entitiesDetected: number | null;
  categories: Record<string, number> | null;
  action: string;
  latencyMs: number | null;
}

const dayAgo = () => new Date(Date.now() - 24 * 3600 * 1000).toISOString();
const nowISO = () => new Date().toISOString();

export function useOverview() {
  const from = dayAgo();
  const to = nowISO();
  return useQuery({
    queryKey: queryKeys.overview(from, to),
    queryFn: () => api.get<Overview>(`/v1/aas/analytics/overview?from=${from}&to=${to}`),
  });
}

export function useUsageSeries(tenantId?: string) {
  const from = dayAgo();
  const to = nowISO();
  const qs = tenantId ? `&tenantId=${tenantId}` : '';
  return useQuery({
    queryKey: queryKeys.usage(from, to, tenantId),
    queryFn: () => api.get<UsagePoint[]>(`/v1/aas/analytics/usage?from=${from}&to=${to}${qs}`),
  });
}

export function useTenants() {
  return useQuery({
    queryKey: queryKeys.tenants(),
    queryFn: () => api.get<Tenant[]>('/v1/aas/tenants'),
  });
}

export function useLogs(tenantId?: string) {
  const qs = tenantId ? `?tenantId=${tenantId}` : '';
  return useQuery({
    queryKey: queryKeys.logs(tenantId),
    queryFn: () => api.get<UsageLog[]>(`/v1/aas/usage-logs${qs}`),
  });
}
