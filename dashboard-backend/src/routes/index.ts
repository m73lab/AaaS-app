import { Router } from 'express';
import { Container } from '../core/dependency-injection/container.js';
import { tenantRoutes } from '../modules/tenants/factories/tenant.bootstrap.js';
import { usageLogRoutes } from '../modules/usage-logs/factories/usage-log.bootstrap.js';
import { analyticsRoutes } from '../modules/analytics/factories/analytics.bootstrap.js';
import { createAuthController } from '../modules/auth/interfaces/http/auth.api.controller.js';
import { SERVICE_KEYS } from '../core/dependency-injection/service-keys.js';

export function createRouter(container: Container): Router {
  const router = Router();

  const authCtrl = createAuthController(container.get(SERVICE_KEYS.AuthService));
  router.post('/auth/login', authCtrl.login);
  router.get('/auth/me', authCtrl.me);

  router.use('/tenants', tenantRoutes(container));
  router.use('/usage-logs', usageLogRoutes(container));
  router.use('/analytics', analyticsRoutes(container));

  return router;
}
