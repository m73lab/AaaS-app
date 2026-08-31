import { createClient, type RedisClientType } from 'redis';
import type { ICacheService } from '../domain/icache.service.js';
import { env } from '../../../config/env.js';

export class RedisCacheService implements ICacheService {
  private client: RedisClientType;

  constructor() {
    this.client = createClient({ url: env.redisUrl });
    this.client.on('error', (err) => console.error('[Redis]', err));
  }

  private async conn(): Promise<RedisClientType> {
    if (!this.client.isOpen) await this.client.connect();
    return this.client;
  }

  async incr(key: string, ttlSeconds: number): Promise<number> {
    const c = await this.conn();
    const count = await c.incr(key);
    if (count === 1) await c.expire(key, ttlSeconds);
    return count;
  }

  async get(key: string): Promise<number> {
    const c = await this.conn();
    const v = await c.get(key);
    return v ? parseInt(v, 10) : 0;
  }

  async set(key: string, value: number, ttlSeconds: number): Promise<void> {
    const c = await this.conn();
    await c.set(key, String(value), { EX: ttlSeconds });
  }

  async del(key: string): Promise<void> {
    const c = await this.conn();
    await c.del(key);
  }

  async ping(): Promise<boolean> {
    try {
      const c = await this.conn();
      const r = await c.ping();
      return r === 'PONG';
    } catch {
      return false;
    }
  }
}
