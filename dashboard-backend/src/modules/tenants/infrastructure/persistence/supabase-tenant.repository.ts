import type { SupabaseClient } from '@supabase/supabase-js';
import { ApiError } from '../../../../lib/errors.js';
import type { TenantEntity } from '../../domain/entities/tenant.entity.js';
import type { TenantRepository } from '../../domain/repositories/tenant.repository.js';

interface Row {
  id: string;
  name: string;
  email: string;
  plan: string;
  rate_limit_per_min: number;
  rate_limit_per_hour: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export class SupabaseTenantRepository implements TenantRepository {
  constructor(private db: SupabaseClient) {}

  private map(r: Row): TenantEntity {
    return {
      id: r.id,
      name: r.name,
      email: r.email,
      plan: r.plan,
      rateLimitPerMin: r.rate_limit_per_min,
      rateLimitPerHour: r.rate_limit_per_hour,
      active: r.active,
      createdAt: r.created_at,
      updatedAt: r.updated_at,
    };
  }

  async findAll(): Promise<TenantEntity[]> {
    const { data, error } = await this.db.from('tenants').select('*').order('created_at', { ascending: false });
    if (error) throw ApiError.internal(error.message);
    return (data as Row[]).map(this.map);
  }

  async findById(id: string): Promise<TenantEntity | null> {
    const { data, error } = await this.db.from('tenants').select('*').eq('id', id).maybeSingle();
    if (error) throw ApiError.internal(error.message);
    return data ? this.map(data as Row) : null;
  }

  async findByApiKeyPrefix(_prefix: string): Promise<TenantEntity | null> {
    return null;
  }

  async create(input: Omit<TenantEntity, 'id' | 'createdAt' | 'updatedAt'>): Promise<TenantEntity> {
    const { data, error } = await this.db
      .from('tenants')
      .insert({
        name: input.name,
        email: input.email,
        plan: input.plan,
        rate_limit_per_min: input.rateLimitPerMin,
        rate_limit_per_hour: input.rateLimitPerHour,
        active: input.active,
      })
      .select()
      .single();
    if (error) {
      if (error.code === '23505') throw ApiError.conflict('Tenant name already exists');
      throw ApiError.internal(error.message);
    }
    return this.map(data as Row);
  }

  async update(id: string, input: Partial<TenantEntity>): Promise<TenantEntity> {
    const patch: Record<string, unknown> = {};
    if (input.name !== undefined) patch.name = input.name;
    if (input.email !== undefined) patch.email = input.email;
    if (input.plan !== undefined) patch.plan = input.plan;
    if (input.rateLimitPerMin !== undefined) patch.rate_limit_per_min = input.rateLimitPerMin;
    if (input.rateLimitPerHour !== undefined) patch.rate_limit_per_hour = input.rateLimitPerHour;
    if (input.active !== undefined) patch.active = input.active;

    const { data, error } = await this.db.from('tenants').update(patch).eq('id', id).select().single();
    if (error) throw ApiError.internal(error.message);
    if (!data) throw ApiError.notFound('Tenant not found');
    return this.map(data as Row);
  }

  async delete(id: string): Promise<void> {
    const { error } = await this.db.from('tenants').delete().eq('id', id);
    if (error) throw ApiError.internal(error.message);
  }
}
