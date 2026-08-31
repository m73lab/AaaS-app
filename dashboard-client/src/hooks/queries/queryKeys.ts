// Stable date windows — computed once at module load, not per render.
const now = new Date();
const dayAgo = new Date(now.getTime() - 24 * 3600 * 1000);
const FROM = dayAgo.toISOString();
const TO = now.toISOString();

export const queryKeys = {
  overview: ['overview', FROM, TO] as const,
  usage: (tenantId?: string) => ['usage', FROM, TO, tenantId] as const,
  logs: ['logs'] as const,
};
