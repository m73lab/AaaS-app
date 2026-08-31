import type { Pool } from 'pg';
import { ApiError } from '../../../lib/errors.js';
import type { AnalyticsRepository, Overview, UsagePoint } from '../domain/analytics.repository.js';

export class PgAnalyticsRepository implements AnalyticsRepository {
  constructor(private db: Pool) {}

  private async fetch(from: string, to: string, tenantId?: string) {
    const conditions = ['timestamp >= $1', 'timestamp <= $2'];
    const values: unknown[] = [from, to];
    let i = 3;
    if (tenantId) { conditions.push(`tenant_id = $${i++}`); values.push(tenantId); }
    const { rows } = await this.db.query(
      `SELECT tenant_id, prompt_tokens, completion_tokens, entities_detected, categories, action, latency_ms, timestamp
       FROM usage_logs WHERE ${conditions.join(' AND ')}`,
      values,
    );
    return rows;
  }

  async getOverview(from: string, to: string): Promise<Overview> {
    const rows = await this.fetch(from, to);
    const tenants = new Set<string>();
    const cats: Record<string, number> = {};
    let prompt = 0, completion = 0, entities = 0, blocked = 0, latSum = 0, latCount = 0;

    for (const r of rows) {
      tenants.add(r.tenant_id);
      prompt += r.prompt_tokens ?? 0;
      completion += r.completion_tokens ?? 0;
      entities += r.entities_detected ?? 0;
      if (r.action === 'blocked' || r.action === 'error') blocked++;
      if (r.latency_ms) { latSum += r.latency_ms; latCount++; }
      if (r.categories) {
        const c = typeof r.categories === 'string' ? JSON.parse(r.categories) : r.categories;
        for (const [k, v] of Object.entries(c as Record<string, number>)) cats[k] = (cats[k] ?? 0) + v;
      }
    }

    const topCategories = Object.entries(cats)
      .map(([category, count]) => ({ category, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 8);

    return {
      totalRequests: rows.length,
      totalPromptTokens: prompt,
      totalCompletionTokens: completion,
      totalEntitiesDetected: entities,
      activeTenants: tenants.size,
      requestsBlocked: blocked,
      avgLatencyMs: latCount ? Math.round(latSum / latCount) : 0,
      topCategories,
    };
  }

  async getUsageSeries(from: string, to: string, tenantId?: string): Promise<UsagePoint[]> {
    const rows = await this.fetch(from, to, tenantId);
    const buckets = new Map<string, UsagePoint>();
    for (const r of rows) {
      const hour = r.timestamp instanceof Date
        ? r.timestamp.toISOString().slice(0, 13)
        : String(r.timestamp).slice(0, 13);
      const key = `${hour}:00`;
      const b = buckets.get(key) ?? { hour: key, requests: 0, tokens: 0, entities: 0 };
      b.requests++;
      b.tokens += (r.prompt_tokens ?? 0) + (r.completion_tokens ?? 0);
      b.entities += r.entities_detected ?? 0;
      buckets.set(key, b);
    }
    return [...buckets.values()].sort((a, b) => a.hour.localeCompare(b.hour));
  }
}
