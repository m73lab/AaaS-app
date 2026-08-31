import crypto from 'crypto';
import { ApiError } from '../../../lib/errors.js';
import { signToken } from '../../../lib/middleware/withAuth.js';
import type { AdminRepository } from '../infrastructure/persistence/supabase-admin.repository.js';
import type { AdminLoginResult } from '../domain/admin.entity.js';

export interface LoginInput {
  email: string;
  password: string;
}

export class AuthService {
  constructor(private adminRepo: AdminRepository) {}

  async login(input: LoginInput): Promise<{ token: string; admin: AdminLoginResult }> {
    const admin = await this.adminRepo.findByEmail(input.email);
    if (!admin) throw ApiError.unauthorized('Invalid credentials');

    const hash = crypto.createHash('sha256').update(input.password).digest('hex');
    if (hash !== admin.passwordHash) throw ApiError.unauthorized('Invalid credentials');

    const token = signToken({
      sub: admin.id,
      email: admin.email,
      tenantId: admin.id,
      role: 'admin',
    });

    return {
      token,
      admin: { id: admin.id, email: admin.email, name: admin.name },
    };
  }
}
