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

export interface AnalyticsRepository {
  getOverview(from: string, to: string): Promise<Overview>;
  getUsageSeries(from: string, to: string, tenantId?: string): Promise<UsagePoint[]>;
}
