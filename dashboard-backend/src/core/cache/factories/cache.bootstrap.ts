import { Container } from '../../dependency-injection/container.js';
import { SERVICE_KEYS } from '../../dependency-injection/service-keys.js';
import { ICacheService } from '../domain/icache.service.js';
import { RedisCacheService } from '../infrastructure/redis-cache.service.js';

export function registerCache(container: Container): void {
  container.register(
    SERVICE_KEYS.CacheService,
    () => new RedisCacheService(),
    { singleton: true },
  );
}
