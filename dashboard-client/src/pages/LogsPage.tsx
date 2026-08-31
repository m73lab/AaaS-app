import { Layout } from '../components/Layout';
import { useLogs } from '../hooks/queries/useDashboard';

function ago(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  return `${Math.floor(m / 60)}h ago`;
}

export function LogsPage() {
  const { data: logs, isLoading } = useLogs();

  return (
    <Layout>
      <div className="page">
        <h1 className="page-title">Usage logs</h1>
        <p className="page-sub">Most recent proxy requests (PII-free).</p>

        {isLoading ? (
          <div className="chart-empty">Loading…</div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>When</th>
                <th>Tenant</th>
                <th>Model</th>
                <th>Format</th>
                <th>Action</th>
                <th>Entities</th>
                <th>Latency</th>
              </tr>
            </thead>
            <tbody>
              {(logs ?? []).map((l) => (
                <tr key={l.id}>
                  <td>{ago(l.timestamp)}</td>
                  <td className="mono">{l.tenantId.slice(0, 8)}</td>
                  <td>{l.model ?? '—'}</td>
                  <td>{l.format ?? '—'}</td>
                  <td>
                    <span className={`badge ${l.action === 'forward' ? 'ok' : l.action === 'blocked' ? 'bad' : 'warn'}`}>
                      {l.action}
                    </span>
                  </td>
                  <td>{l.entitiesDetected ?? '—'}</td>
                  <td>{l.latencyMs ? `${l.latencyMs} ms` : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </Layout>
  );
}
