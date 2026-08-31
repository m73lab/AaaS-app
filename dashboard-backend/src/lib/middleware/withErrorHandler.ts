import type { Request, Response, NextFunction } from 'express';
import { ZodSchema, type ZodIssue } from 'zod';
import { ApiError } from '../errors.js';

export function withErrorHandler(
  fn: (req: Request, res: Response, next: NextFunction) => unknown,
) {
  return async (req: Request, res: Response, next: NextFunction) => {
    try {
      await fn(req, res, next);
    } catch (err) {
      if (err instanceof ApiError) {
        res.status(err.status).json({ error: { code: err.code, message: err.message } });
        return;
      }
      if (err && typeof err === 'object' && 'code' in err && (err as { code?: string }).code === 'PGRST301') {
        res.status(401).json({ error: { code: 'UNAUTHORIZED', message: 'Invalid or expired token' } });
        return;
      }
      console.error('[Unhandled]', err);
      res.status(500).json({ error: { code: 'INTERNAL', message: 'Internal server error' } });
    }
  };
}

export function withValidation<T>(schema: ZodSchema<T>) {
  return (handler: (req: Request, res: Response, next: NextFunction) => unknown) =>
    (req: Request, res: Response, next: NextFunction) => {
      const result = schema.safeParse(req.body);
      if (!result.success) {
        next(ApiError.badRequest(result.error.issues.map((e: ZodIssue) => e.message).join(', ')));
        return;
      }
      req.body = result.data;
      return handler(req, res, next);
    };
}
