import type { TenantEntity } from '../entities/tenant.entity.js';

export interface TenantRepository {
  findAll(): Promise<TenantEntity[]>;
  findById(id: string): Promise<TenantEntity | null>;
  findByApiKeyPrefix(prefix: string): Promise<TenantEntity | null>;
  create(input: Omit<TenantEntity, 'id' | 'createdAt' | 'updatedAt'>): Promise<TenantEntity>;
  update(id: string, input: Partial<TenantEntity>): Promise<TenantEntity>;
  delete(id: string): Promise<void>;
}
