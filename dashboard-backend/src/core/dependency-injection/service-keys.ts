export const SERVICE_KEYS = {
  Pool: Symbol('Pool'),
  TenantRepository: Symbol('TenantRepository'),
  TenantService: Symbol('TenantService'),
  UsageLogRepository: Symbol('UsageLogRepository'),
  UsageLogService: Symbol('UsageLogService'),
  AnalyticsService: Symbol('AnalyticsService'),
  AuthService: Symbol('AuthService'),
  AdminRepository: Symbol('AdminRepository'),
} as const;

export type ServiceKey = (typeof SERVICE_KEYS)[keyof typeof SERVICE_KEYS];
