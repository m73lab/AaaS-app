import type { Pool } from 'pg';
import { ApiError } from '../../../../lib/errors.js';
import type { TenantEntity } from '../../domain/entities/tenant.entity.js';
import type { TenantRepository } from '../../domain/repositories/tenant.repository.js';

export class PgTenantRepository implements TenantRepository {
  constructor(private db: Pool) {}

  async findAll(): Promise<TenantEntity[]> {
    const { rows } = await this.db.query(
      'SELECT * FROM tenants ORDER BY created_at DESC',
    );
    return rows.map((r) => ({
      id: r.id,
      name: r.name,
      email: r.email,
      plan: r.plan,
      rateLimitPerMin: r.rate_limit_per_min,
      rateLimitPerHour: r.rate_limit_per_hour,
      active: r.active,
      createdAt: r.created_at,
      updatedAt: r.updated_at,
    }));
  }

  async findById(id: string): Promise<TenantEntity | null> {
    const { rows, rowCount } = await this.db.query(
      'SELECT * FROM tenants WHERE id = $1',
      [id],
    );
    if (!rowCount) return null;
    const r = rows[0];
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

  async findByApiKeyPrefix(_prefix: string): Promise<TenantEntity | null> {
    return null;
  }

  async create(input: Omit<TenantEntity, 'id' | 'createdAt' | 'updatedAt'>): Promise<TenantEntity> {
    const { rows, rowCount } = await this.db.query(
      `INSERT INTO tenants (name, email, plan, rate_limit_per_min, rate_limit_per_hour, active)
       VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
      [input.name, input.email, input.plan, input.rateLimitPerMin, input.rateLimitPerHour, input.active],
    );
    if (!rowCount) throw ApiError.internal('Failed to create tenant');
    const r = rows[0];
    return {
      id: r.id, name: r.name, email: r.email, plan: r.plan,
      rateLimitPerMin: r.rate_limit_per_min, rateLimitPerHour: r.rate_limit_per_hour,
      active: r.active, createdAt: r.created_at, updatedAt: r.updated_at,
    };
  }

  async update(id: string, input: Partial<TenantEntity>): Promise<TenantEntity> {
    const fields: string[] = [];
    const values: unknown[] = [];
    let i = 1;
    if (input.name !== undefined) { fields.push(`name = $${i++}`); values.push(input.name); }
    if (input.email !== undefined) { fields.push(`email = $${i++}`); values.push(input.email); }
    if (input.plan !== undefined) { fields.push(`plan = $${i++}`); values.push(input.plan); }
    if (input.rateLimitPerMin !== undefined) { fields.push(`rate_limit_per_min = $${i++}`); values.push(input.rateLimitPerMin); }
    if (input.rateLimitPerHour !== undefined) { fields.push(`rate_limit_per_hour = $${i++}`); values.push(input.rateLimitPerHour); }
    if (input.active !== undefined) { fields.push(`active = $${i++}`); values.push(input.active); }
    values.push(id);
    const { rows, rowCount } = await this.db.query(
      `UPDATE tenants SET ${fields.join(', ')} WHERE id = $${i} RETURNING *`,
      values,
    );
    if (!rowCount) throw ApiError.notFound('Tenant not found');
    const r = rows[0];
    return {
      id: r.id, name: r.name, email: r.email, plan: r.plan,
      rateLimitPerMin: r.rate_limit_per_min, rateLimitPerHour: r.rate_limit_per_hour,
      active: r.active, createdAt: r.created_at, updatedAt: r.updated_at,
    };
  }

  async delete(id: string): Promise<void> {
    await this.db.query('DELETE FROM tenants WHERE id = $1', [id]);
  }
}
