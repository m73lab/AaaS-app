export const queryKeys = {
  overview: (from: string, to: string) => ['overview', from, to] as const,
  usage: (from: string, to: string, tenantId?: string) => ['usage', from, to, tenantId] as const,
  tenants: () => ['tenants'] as const,
  logs: (tenantId?: string) => ['logs', tenantId] as const,
};
