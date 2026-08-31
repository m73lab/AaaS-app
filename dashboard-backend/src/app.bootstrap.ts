import { Container } from './core/dependency-injection/container.js';
import { SERVICE_KEYS } from './core/dependency-injection/service-keys.js';
import { pool } from './config/pg.js';
import { registerTenant } from './modules/tenants/factories/tenant.bootstrap.js';
import { registerUsageLog } from './modules/usage-logs/factories/usage-log.bootstrap.js';
import { registerAnalytics } from './modules/analytics/factories/analytics.bootstrap.js';
import { registerAuth } from './modules/auth/factories/auth.bootstrap.js';

export function buildContainer(): Container {
  const c = new Container();
  c.register(SERVICE_KEYS.Pool, () => pool, { singleton: true });
  registerTenant(c);
  registerUsageLog(c);
  registerAnalytics(c);
  registerAuth(c);
  return c;
}
