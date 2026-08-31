import type { UsagePoint } from '../hooks/queries/useDashboard';

export function UsageChart({ data }: { data: UsagePoint[] }) {
  if (!data || data.length === 0) {
    return <div className="chart-empty">No usage data in the selected window.</div>;
  }
  const max = Math.max(...data.map((d) => d.requests), 1);
  const width = 640;
  const height = 200;
  const gap = 4;
  const barW = (width - gap * (data.length - 1)) / data.length;
  const labelEvery = Math.ceil(data.length / 8);

  return (
    <svg className="usage-chart" viewBox={`0 0 ${width} ${height + 24}`} role="img" aria-label="Usage chart">
      {data.map((d, i) => {
        const h = Math.round((d.requests / max) * height);
        const x = i * (barW + gap);
        return (
          <g key={d.hour}>
            <rect x={x} y={height - h} width={barW} height={h} className="bar" rx={2} />
            {i % labelEvery === 0 && (
              <text x={x} y={height + 16} className="bar-label">
                {d.hour.slice(5, 16).replace('T', ' ')}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}
