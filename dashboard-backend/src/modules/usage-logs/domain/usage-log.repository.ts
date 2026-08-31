import type { UsageLogEntity } from '../domain/usage-log.entity.js';

export interface IngestUsageInput {
  tenantId: string;
  sessionId: string;
  model?: string;
  provider?: string;
  format?: string;
  promptTokens?: number;
  completionTokens?: number;
  entitiesDetected?: number;
  categories?: Record<string, number>;
  action: string;
  latencyMs?: number;
}

export interface UsageQuery {
  tenantId?: string;
  from?: string;
  to?: string;
  limit?: number;
}

export interface UsageLogRepository {
  insert(input: IngestUsageInput): Promise<void>;
  query(q: UsageQuery): Promise<UsageLogEntity[]>;
}
