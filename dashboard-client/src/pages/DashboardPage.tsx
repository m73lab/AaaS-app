import { Layout } from '../components/Layout';
import { MetricCard } from '../components/MetricCard';
import { UsageChart } from '../components/UsageChart';
import { useOverview, useUsageSeries } from '../hooks/queries/useDashboard';

export function DashboardPage() {
  const { data: overview, isLoading } = useOverview();
  const { data: series } = useUsageSeries();

  return (
    <Layout>
      <div className="page">
        <h1 className="page-title">Overview</h1>
        <p className="page-sub">Last 24 hours across all tenants.</p>

        <div className="metric-grid">
          <MetricCard label="Requests" value={overview?.totalRequests ?? '—'} />
          <MetricCard label="Entities detected" value={overview?.totalEntitiesDetected ?? '—'} />
          <MetricCard label="Active tenants" value={overview?.activeTenants ?? '—'} />
          <MetricCard label="Blocked" value={overview?.requestsBlocked ?? '—'} sub="policy violations" />
          <MetricCard
            label="Avg latency"
            value={overview ? `${overview.avgLatencyMs} ms` : '—'}
          />
          <MetricCard
            label="Tokens"
            value={
              overview
                ? (overview.totalPromptTokens + overview.totalCompletionTokens).toLocaleString()
                : '—'
            }
            sub="prompt + completion"
          />
        </div>

        <div className="panel">
          <h2>Requests per hour</h2>
          {isLoading ? <div className="chart-empty">Loading…</div> : <UsageChart data={series ?? []} />}
        </div>

        {overview && overview.topCategories.length > 0 && (
          <div className="panel">
            <h2>Top detected categories</h2>
            <div className="cat-list">
              {overview.topCategories.map((c) => (
                <div key={c.category} className="cat-row">
                  <span className="cat-name">{c.category}</span>
                  <span className="cat-count">{c.count}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
