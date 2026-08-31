import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useClientSession } from '../context/ClientSessionContext';

const NAV = [
  { to: '/dashboard', label: 'Dashboard' },
  { to: '/logs', label: 'Logs' },
  { to: '/docs', label: 'API Docs' },
];

export function Layout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useClientSession();
  const loc = useLocation();
  const nav = useNavigate();

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">A</span>
          <span className="brand-name">AaaS</span>
        </div>
        <nav className="side-nav">
          {NAV.map((n) => (
            <Link
              key={n.to}
              to={n.to}
              className={loc.pathname.startsWith(n.to) ? 'side-link active' : 'side-link'}
            >
              {n.label}
            </Link>
          ))}
        </nav>
        <div className="side-foot">
          <div className="side-user">{user?.email}</div>
          <button
            className="btn-ghost"
            onClick={() => {
              logout();
              nav('/login');
            }}
          >
            Sign out
          </button>
        </div>
      </aside>
      <main className="content">{children}</main>
    </div>
  );
}
