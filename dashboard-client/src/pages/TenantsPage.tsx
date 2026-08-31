import { Layout } from '../components/Layout';
import { useTenants } from '../hooks/queries/useDashboard';

export function TenantsPage() {
  const { data: tenants, isLoading } = useTenants();

  return (
    <Layout>
      <div className="page">
        <h1 className="page-title">Tenants</h1>
        <p className="page-sub">Rate-limited consumers of the anonymization proxy.</p>

        {isLoading ? (
          <div className="chart-empty">Loading…</div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Plan</th>
                <th>Limit / min</th>
                <th>Limit / hour</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {(tenants ?? []).map((t) => (
                <tr key={t.id}>
                  <td>{t.name}</td>
                  <td>{t.email}</td>
                  <td>{t.plan}</td>
                  <td>{t.rateLimitPerMin}</td>
                  <td>{t.rateLimitPerHour}</td>
                  <td>
                    <span className={t.active ? 'badge ok' : 'badge off'}>
                      {t.active ? 'active' : 'disabled'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </Layout>
  );
}
