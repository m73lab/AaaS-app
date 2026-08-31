import type { SupabaseClient } from '@supabase/supabase-js';
import { ApiError } from '../../../lib/errors.js';
import type { UsageLogEntity } from '../domain/usage-log.entity.js';
import type { IngestUsageInput, UsageLogRepository, UsageQuery } from '../domain/usage-log.repository.js';

interface Row {
  id: number;
  tenant_id: string;
  timestamp: string;
  session_id: string;
  model: string | null;
  provider: string | null;
  format: string | null;
  prompt_tokens: number | null;
  completion_tokens: number | null;
  entities_detected: number | null;
  categories: Record<string, number> | null;
  action: string;
  latency_ms: number | null;
}

export class SupabaseUsageLogRepository implements UsageLogRepository {
  constructor(private db: SupabaseClient) {}

  async insert(input: IngestUsageInput): Promise<void> {
    const { error } = await this.db.from('usage_logs').insert({
      tenant_id: input.tenantId,
      session_id: input.sessionId,
      model: input.model ?? null,
      provider: input.provider ?? null,
      format: input.format ?? null,
      prompt_tokens: input.promptTokens ?? null,
      completion_tokens: input.completionTokens ?? null,
      entities_detected: input.entitiesDetected ?? null,
      categories: input.categories ?? null,
      action: input.action,
      latency_ms: input.latencyMs ?? null,
    });
    if (error) throw ApiError.internal(error.message);
  }

  async query(q: UsageQuery): Promise<UsageLogEntity[]> {
    let query = this.db.from('usage_logs').select('*');
    if (q.tenantId) query = query.eq('tenant_id', q.tenantId);
    if (q.from) query = query.gte('timestamp', q.from);
    if (q.to) query = query.lte('timestamp', q.to);
    query = query.order('timestamp', { ascending: false }).limit(q.limit ?? 100);

    const { data, error } = await query;
    if (error) throw ApiError.internal(error.message);
    return (data as Row[]).map((r) => ({
      id: r.id,
      tenantId: r.tenant_id,
      timestamp: r.timestamp,
      sessionId: r.session_id,
      model: r.model,
      provider: r.provider,
      format: r.format,
      promptTokens: r.prompt_tokens,
      completionTokens: r.completion_tokens,
      entitiesDetected: r.entities_detected,
      categories: r.categories,
      action: r.action,
      latencyMs: r.latency_ms,
    }));
  }
}
