import type { SupabaseClient } from '@supabase/supabase-js';
import { ApiError } from '../../../../lib/errors.js';
import type { AdminEntity } from '../../domain/admin.entity.js';

export interface AdminRepository {
  findByEmail(email: string): Promise<AdminEntity | null>;
}

export class SupabaseAdminRepository implements AdminRepository {
  constructor(private db: SupabaseClient) {}

  async findByEmail(email: string): Promise<AdminEntity | null> {
    const { data, error } = await this.db
      .from('admins')
      .select('id, email, password_hash, name, created_at')
      .eq('email', email)
      .maybeSingle();

    if (error) throw ApiError.internal(error.message);
    if (!data) return null;

    return {
      id: data.id,
      email: data.email,
      passwordHash: data.password_hash,
      name: data.name,
      createdAt: data.created_at,
    };
  }
}
