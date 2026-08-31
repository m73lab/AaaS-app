import type { Pool } from 'pg';
import { ApiError } from '../../../../lib/errors.js';
import type { AdminEntity } from '../../domain/admin.entity.js';

export interface AdminRepository {
  findByEmail(email: string): Promise<AdminEntity | null>;
}

export class PgAdminRepository implements AdminRepository {
  constructor(private db: Pool) {}

  async findByEmail(email: string): Promise<AdminEntity | null> {
    const { rows, rowCount } = await this.db.query(
      'SELECT id, email, password_hash, name, created_at FROM admins WHERE email = $1',
      [email],
    );
    if (!rowCount) return null;
    const r = rows[0];
    return {
      id: r.id,
      email: r.email,
      passwordHash: r.password_hash,
      name: r.name,
      createdAt: r.created_at,
    };
  }
}
