export interface TenantEntity {
  id: string;
  name: string;
  email: string;
  plan: string;
  rateLimitPerMin: number;
  rateLimitPerHour: number;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}
