import type { IngestUsageInput, UsageLogRepository, UsageQuery } from '../domain/usage-log.repository.js';

export class UsageLogService {
  constructor(private repo: UsageLogRepository) {}

  async ingest(input: IngestUsageInput): Promise<void> {
    await this.repo.insert(input);
  }

  async query(q: UsageQuery) {
    return this.repo.query(q);
  }
}
