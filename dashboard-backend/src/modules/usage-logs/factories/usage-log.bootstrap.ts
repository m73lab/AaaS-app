import { Router } from 'express';
import { Container } from '../../../core/dependency-injection/container.js';
import { SERVICE_KEYS } from '../../../core/dependency-injection/service-keys.js';
import { PgUsageLogRepository } from '../infrastructure/supabase-usage-log.repository.js';
import { UsageLogService } from '../application/usage-log.service.js';
import { createUsageLogController } from '../interfaces/http/usage-log.api.controller.js';

export function registerUsageLog(container: Container): void {
  container.register(SERVICE_KEYS.UsageLogRepository, (c) =>
    new PgUsageLogRepository(c.get(SERVICE_KEYS.Pool)),
  );
  container.register(SERVICE_KEYS.UsageLogService, (c) =>
    new UsageLogService(c.get(SERVICE_KEYS.UsageLogRepository)),
  );
}

export function usageLogRoutes(container: Container): Router {
  const router = Router();
  const controller = createUsageLogController(container.get(SERVICE_KEYS.UsageLogService));
  router.post('/', controller.ingest);
  router.get('/', controller.list);
  return router;
}
