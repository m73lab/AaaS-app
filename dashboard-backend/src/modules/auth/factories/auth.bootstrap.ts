import { Container } from '../../../core/dependency-injection/container.js';
import { SERVICE_KEYS } from '../../../core/dependency-injection/service-keys.js';
import { PgAdminRepository } from '../infrastructure/persistence/supabase-admin.repository.js';
import { AuthService } from '../application/auth.service.js';

export function registerAuth(container: Container): void {
  container.register(SERVICE_KEYS.AdminRepository, (c) =>
    new PgAdminRepository(c.get(SERVICE_KEYS.Pool)),
  );
  container.register(SERVICE_KEYS.AuthService, (c) =>
    new AuthService(c.get(SERVICE_KEYS.AdminRepository)),
  );
}
