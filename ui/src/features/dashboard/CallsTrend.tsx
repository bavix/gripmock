import { useMemo } from 'react';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Legend } from 'recharts';
import { useStore } from '../../lib/store';
import { bucketCalls } from './buckets';
import type { CallRecord } from '../../lib/types';

// Series colors validated with the dataviz palette checker for BOTH surfaces:
// accent (calls) + #e5484d (errors, status color w/ legend label).
const SERIES = {
  light: { ok: '#4f6bed', errors: '#e5484d', grid: '#e4e8ee', text: '#8a94a6' },
  dark: { ok: '#6d86f5', errors: '#e5484d', grid: '#263349', text: '#6b7890' },
};

export default function CallsTrend({ records }: { records: CallRecord[] }) {
  const theme = useStore((s) => s.theme);
  const c = SERIES[theme === 'dark' ? 'dark' : 'light'];
  const data = useMemo(() => bucketCalls(records), [records]);
  const empty = records.length === 0;

  return (
    <div className="card">
      <div className="card-header">Calls per minute</div>
      <div style={{ padding: '10px 6px 4px', height: 180 }}>
        {empty ? (
          <div className="empty" style={{ padding: 24 }}>No calls recorded yet.</div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 4, right: 12, left: -18, bottom: 0 }}>
              <CartesianGrid stroke={c.grid} strokeDasharray="0" vertical={false} />
              <XAxis dataKey="label" tick={{ fontSize: 10, fill: c.text }} tickLine={false} axisLine={{ stroke: c.grid }} minTickGap={28} />
              <YAxis allowDecimals={false} tick={{ fontSize: 10, fill: c.text }} tickLine={false} axisLine={false} width={34} />
              <Tooltip
                cursor={{ stroke: c.text, strokeWidth: 1 }}
                contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border)', borderRadius: 8, fontSize: 12, color: 'var(--text)' }}
                labelStyle={{ color: 'var(--text-muted)' }}
              />
              <Legend wrapperStyle={{ fontSize: 11.5 }} iconType="plainline" />
              <Area type="monotone" dataKey="ok" name="OK" stroke={c.ok} strokeWidth={2} fill={c.ok} fillOpacity={0.12} dot={false} isAnimationActive={false} />
              <Area type="monotone" dataKey="errors" name="Errors" stroke={c.errors} strokeWidth={2} fill={c.errors} fillOpacity={0.12} dot={false} isAnimationActive={false} />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
