import type { Request, Response, NextFunction } from 'express';
import { ICacheService } from '../../core/cache/domain/icache.service.js';
import { ApiError } from '../errors.js';

export interface RateLimitOptions {
  windowSeconds: number;
  max: number | ((req: Request) => number);
  key: (req: Request) => string;
  message?: string;
}

export function withRateLimit(cache: ICacheService, opts: RateLimitOptions) {
  return async (req: Request, _res: Response, next: NextFunction) => {
    const key = `ratelimit:${opts.key(req)}`;
    const max = typeof opts.max === 'function' ? opts.max(req) : opts.max;
    try {
      const count = await cache.incr(key, opts.windowSeconds);
      if (count > max) {
        next(ApiError.tooManyRequests(opts.message ?? 'Rate limit exceeded'));
        return;
      }
      next();
    } catch (err) {
      console.error('[RateLimit] cache error, allowing:', err);
      next();
    }
  };
}
