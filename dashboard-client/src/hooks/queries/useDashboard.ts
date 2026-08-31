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

const now = new Date();
const dayAgo = new Date(now.getTime() - 24 * 3600 * 1000);
const FROM = dayAgo.toISOString();
const TO = now.toISOString();

export function useOverview() {
  return useQuery({
    queryKey: queryKeys.overview,
    queryFn: () => api.get<Overview>(`/v1/aas/analytics/overview?from=${FROM}&to=${TO}`),
    refetchInterval: 30000,
  });
}

export function useUsageSeries() {
  return useQuery({
    queryKey: queryKeys.usage(),
    queryFn: () => api.get<UsagePoint[]>(`/v1/aas/analytics/usage?from=${FROM}&to=${TO}`),
    refetchInterval: 30000,
  });
}

export function useLogs() {
  return useQuery({
    queryKey: queryKeys.logs,
    queryFn: () => api.get<UsageLog[]>('/v1/aas/usage-logs'),
    refetchInterval: 10000,
  });
}
