import { Container } from '../../../core/dependency-injection/container.js';
import { SERVICE_KEYS } from '../../../core/dependency-injection/service-keys.js';
import { SupabaseAdminRepository } from '../infrastructure/persistence/supabase-admin.repository.js';
import { AuthService } from '../application/auth.service.js';
import { createAuthController } from '../interfaces/http/auth.api.controller.js';

export function registerAuth(container: Container): void {
  container.register(SERVICE_KEYS.ApiKeyRepository, (c) =>
    new SupabaseAdminRepository(c.get(SERVICE_KEYS.SupabaseClient)),
  );
  container.register(SERVICE_KEYS.AuthService, (c) =>
    new AuthService(c.get(SERVICE_KEYS.ApiKeyRepository)),
  );
}
