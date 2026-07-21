import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';
import { useStore } from '../../lib/store';

// Status semantics (covered = good, uncovered = warning); labels carry
// identity alongside color per dataviz rules.
const COLORS = {
  light: { covered: '#16a34a', uncovered: '#d97706' },
  dark: { covered: '#35c46a', uncovered: '#f0a338' },
};

export default function CoverageDonut({ covered, total }: { covered: number; total: number }) {
  const theme = useStore((s) => s.theme);
  const c = COLORS[theme === 'dark' ? 'dark' : 'light'];
  const uncovered = Math.max(0, total - covered);
  const pct = total > 0 ? Math.round((covered / total) * 100) : 0;
  const data = [
    { name: 'Covered', value: covered, color: c.covered },
    { name: 'No stubs', value: uncovered, color: c.uncovered },
  ].filter((d) => d.value > 0);

  return (
    <div className="card">
      <div className="card-header">Method coverage</div>
      <div style={{ padding: 10, height: 180, position: 'relative' }}>
        {total === 0 ? (
          <div className="empty" style={{ padding: 24 }}>No services loaded.</div>
        ) : (
          <>
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={data} dataKey="value" nameKey="name" innerRadius="62%" outerRadius="85%"
                  paddingAngle={data.length > 1 ? 3 : 0} stroke="var(--bg-secondary)" strokeWidth={2} isAnimationActive={false}>
                  {data.map((d) => <Cell key={d.name} fill={d.color} />)}
                </Pie>
                <Tooltip contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border)', borderRadius: 8, fontSize: 12, color: 'var(--text)' }} />
              </PieChart>
            </ResponsiveContainer>
            {/* Hero number in the hole + explicit labels (identity not color-alone) */}
            <div style={{ position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', pointerEvents: 'none' }}>
              <span style={{ fontSize: 22, fontWeight: 700 }}>{pct}%</span>
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{covered}/{total} methods</span>
            </div>
            <div style={{ position: 'absolute', bottom: 6, left: 0, right: 0, display: 'flex', justifyContent: 'center', gap: 14, fontSize: 11, color: 'var(--text-secondary)' }}>
              <span><span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: 2, background: c.covered, marginRight: 4 }} />Covered</span>
              {uncovered > 0 && <span><span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: 2, background: c.uncovered, marginRight: 4 }} />No stubs</span>}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
