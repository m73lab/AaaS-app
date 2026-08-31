export interface UsageLogEntity {
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
