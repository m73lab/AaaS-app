export const SERVICE_KEYS = {
  SupabaseClient: Symbol('SupabaseClient'),
  CacheService: Symbol('CacheService'),
  EventBus: Symbol('EventBus'),

  TenantRepository: Symbol('TenantRepository'),
  TenantService: Symbol('TenantService'),

  UsageLogRepository: Symbol('UsageLogRepository'),
  UsageLogService: Symbol('UsageLogService'),

  AnalyticsService: Symbol('AnalyticsService'),

  AuthService: Symbol('AuthService'),
  ApiKeyRepository: Symbol('ApiKeyRepository'),

  RateLimitMiddleware: Symbol('RateLimitMiddleware'),
  AuthMiddleware: Symbol('AuthMiddleware'),
} as const;

export type ServiceKey = (typeof SERVICE_KEYS)[keyof typeof SERVICE_KEYS];
