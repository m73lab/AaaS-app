import type { SupabaseClient } from '@supabase/supabase-js';
import { ApiError } from '../../../lib/errors.js';
import type { AnalyticsRepository, Overview, UsagePoint } from '../domain/analytics.repository.js';

interface LogRow {
  tenant_id: string;
  prompt_tokens: number | null;
  completion_tokens: number | null;
  entities_detected: number | null;
  categories: Record<string, number> | null;
  action: string;
  latency_ms: number | null;
  timestamp: string;
}

export class SupabaseAnalyticsRepository implements AnalyticsRepository {
  constructor(private db: SupabaseClient) {}

  private async fetch(from: string, to: string, tenantId?: string): Promise<LogRow[]> {
    let q = this.db
      .from('usage_logs')
      .select('tenant_id, prompt_tokens, completion_tokens, entities_detected, categories, action, latency_ms, timestamp')
      .gte('timestamp', from)
      .lte('timestamp', to);
    if (tenantId) q = q.eq('tenant_id', tenantId);
    const { data, error } = await q;
    if (error) throw ApiError.internal(error.message);
    return (data as LogRow[]) ?? [];
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
      if (r.categories) for (const [k, v] of Object.entries(r.categories)) cats[k] = (cats[k] ?? 0) + v;
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
      const hour = r.timestamp.slice(0, 13); // YYYY-MM-DDTHH
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
