import { Router } from 'express';
import { Container } from '../../../core/dependency-injection/container.js';
import { SERVICE_KEYS } from '../../../core/dependency-injection/service-keys.js';
import { PgAnalyticsRepository } from '../infrastructure/supabase-analytics.repository.js';
import { AnalyticsService } from '../application/analytics.service.js';
import { createAnalyticsController } from '../interfaces/http/analytics.api.controller.js';

export function registerAnalytics(container: Container): void {
  container.register(SERVICE_KEYS.AnalyticsService, (c) =>
    new AnalyticsService(new PgAnalyticsRepository(c.get(SERVICE_KEYS.Pool))),
  );
}

export function analyticsRoutes(container: Container): Router {
  const router = Router();
  const controller = createAnalyticsController(container.get(SERVICE_KEYS.AnalyticsService));
  router.get('/overview', controller.overview);
  router.get('/usage', controller.series);
  return router;
}
