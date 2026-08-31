import type { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import { env } from '../../config/env.js';
import { ApiError } from '../errors.js';

export interface AuthPayload {
  sub: string;
  email: string;
  tenantId: string;
  role: 'admin' | 'user';
}

declare global {
  namespace Express {
    interface Request {
      auth?: AuthPayload;
    }
  }
}

export function signToken(payload: Omit<AuthPayload, 'iat' | 'exp'>): string {
  return jwt.sign(payload, env.jwtSecret, { expiresIn: '7d' });
}

export function withAuth(opts: { requireAdmin?: boolean } = {}) {
  return (handler: (req: Request, res: Response, next: NextFunction) => unknown) =>
    (req: Request, res: Response, next: NextFunction) => {
      const header = req.headers.authorization;
      if (!header || !header.startsWith('Bearer ')) {
        next(ApiError.unauthorized('Missing bearer token'));
        return;
      }
      try {
        const decoded = jwt.verify(header.slice(7), env.jwtSecret) as AuthPayload;
        req.auth = decoded;
        if (opts.requireAdmin && decoded.role !== 'admin') {
          next(ApiError.forbidden('Admin role required'));
          return;
        }
        return handler(req, res, next);
      } catch {
        next(ApiError.unauthorized('Invalid token'));
      }
    };
}
