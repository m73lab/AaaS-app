import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useClientSession } from '../context/ClientSessionContext';
import { useToast } from '../context/ToastContext';

export function LoginPage() {
  const { login } = useClientSession();
  const { showToast } = useToast();
  const nav = useNavigate();
  const [email, setEmail] = useState('admin@aas.com');
  const [password, setPassword] = useState('admin123');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await login(email, password);
      nav('/dashboard');
    } catch (err) {
      showToast(err instanceof Error ? err.message : 'Login failed', 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-wrap">
      <form className="login-card" onSubmit={submit}>
        <div className="brand">
          <span className="brand-mark">A</span>
          <span className="brand-name">AaaS</span>
        </div>
        <h2>Sign in to your dashboard</h2>
        <label className="field">
          <span>Email</span>
          <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
        </label>
        <label className="field">
          <span>Password</span>
          <input
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            type="password"
            required
          />
        </label>
        <button className="btn-primary btn-lg" disabled={loading}>
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  );
}
