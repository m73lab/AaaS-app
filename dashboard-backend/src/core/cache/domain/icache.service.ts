export interface ICacheService {
  incr(key: string, ttlSeconds: number): Promise<number>;
  get(key: string): Promise<number>;
  set(key: string, value: number, ttlSeconds: number): Promise<void>;
  del(key: string): Promise<void>;
  ping(): Promise<boolean>;
}
