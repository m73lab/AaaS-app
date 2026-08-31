import { Router } from 'express';
import { Container } from '../../../core/dependency-injection/container.js';
import { SERVICE_KEYS } from '../../../core/dependency-injection/service-keys.js';
import { SupabaseTenantRepository } from '../infrastructure/persistence/supabase-tenant.repository.js';
import { TenantService } from '../application/tenant.service.js';
import { createTenantController } from '../interfaces/http/tenant.api.controller.js';

export function registerTenant(container: Container): void {
  container.register(SERVICE_KEYS.TenantRepository, (c) =>
    new SupabaseTenantRepository(c.get(SERVICE_KEYS.SupabaseClient)),
  );
  container.register(SERVICE_KEYS.TenantService, (c) =>
    new TenantService(c.get(SERVICE_KEYS.TenantRepository)),
  );
}

export function tenantRoutes(container: Container): Router {
  const router = Router();
  const controller = createTenantController(container.get(SERVICE_KEYS.TenantService));
  router.get('/', controller.list);
  router.get('/:id', controller.get);
  router.post('/', controller.create);
  router.patch('/:id', controller.update);
  router.delete('/:id', controller.remove);
  return router;
}
