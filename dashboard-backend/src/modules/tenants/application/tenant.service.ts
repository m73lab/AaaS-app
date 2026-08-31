import { ApiError } from '../../../lib/errors.js';
import type { TenantEntity } from '../domain/entities/tenant.entity.js';
import type { TenantRepository } from '../domain/repositories/tenant.repository.js';

export interface CreateTenantInput {
  name: string;
  email: string;
  plan?: string;
  rateLimitPerMin?: number;
  rateLimitPerHour?: number;
}

export interface UpdateTenantInput {
  name?: string;
  email?: string;
  plan?: string;
  rateLimitPerMin?: number;
  rateLimitPerHour?: number;
  active?: boolean;
}

export class TenantService {
  constructor(private repo: TenantRepository) {}

  async list(): Promise<TenantEntity[]> {
    return this.repo.findAll();
  }

  async getById(id: string): Promise<TenantEntity> {
    const t = await this.repo.findById(id);
    if (!t) throw ApiError.notFound('Tenant not found');
    return t;
  }

  async create(input: CreateTenantInput): Promise<TenantEntity> {
    return this.repo.create({
      name: input.name,
      email: input.email,
      plan: input.plan ?? 'free',
      rateLimitPerMin: input.rateLimitPerMin ?? 100,
      rateLimitPerHour: input.rateLimitPerHour ?? 1000,
      active: true,
    });
  }

  async update(id: string, input: UpdateTenantInput): Promise<TenantEntity> {
    return this.repo.update(id, input);
  }

  async remove(id: string): Promise<void> {
    await this.repo.delete(id);
  }
}
