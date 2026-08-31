import type { Pool } from 'pg';
import { ApiError } from '../../../lib/errors.js';
import type { UsageLogEntity } from '../domain/usage-log.entity.js';
import type { IngestUsageInput, UsageLogRepository, UsageQuery } from '../domain/usage-log.repository.js';

export class PgUsageLogRepository implements UsageLogRepository {
  constructor(private db: Pool) {}

  async insert(input: IngestUsageInput): Promise<void> {
    await this.db.query(
      `INSERT INTO usage_logs (tenant_id, session_id, model, provider, format, prompt_tokens, completion_tokens, entities_detected, categories, action, latency_ms)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
      [
        input.tenantId, input.sessionId, input.model ?? null, input.provider ?? null,
        input.format ?? null, input.promptTokens ?? null, input.completionTokens ?? null,
        input.entitiesDetected ?? null,
        input.categories ? JSON.stringify(input.categories) : null,
        input.action, input.latencyMs ?? null,
      ],
    );
  }

  async query(q: UsageQuery): Promise<UsageLogEntity[]> {
    const conditions: string[] = [];
    const values: unknown[] = [];
    let i = 1;
    if (q.tenantId) { conditions.push(`tenant_id = $${i++}`); values.push(q.tenantId); }
    if (q.from) { conditions.push(`timestamp >= $${i++}`); values.push(q.from); }
    if (q.to) { conditions.push(`timestamp <= $${i++}`); values.push(q.to); }
    const where = conditions.length ? `WHERE ${conditions.join(' AND ')}` : '';
    const limit = q.limit ?? 100;
    const { rows } = await this.db.query(
      `SELECT * FROM usage_logs ${where} ORDER BY timestamp DESC LIMIT $${i}`,
      [...values, limit],
    );
    return rows.map((r) => ({
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
