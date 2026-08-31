import type { AnalyticsRepository, Overview, UsagePoint } from '../domain/analytics.repository.js';

export class AnalyticsService {
  constructor(private repo: AnalyticsRepository) {}

  async overview(from: string, to: string): Promise<Overview> {
    return this.repo.getOverview(from, to);
  }

  async usageSeries(from: string, to: string, tenantId?: string): Promise<UsagePoint[]> {
    return this.repo.getUsageSeries(from, to, tenantId);
  }
}
