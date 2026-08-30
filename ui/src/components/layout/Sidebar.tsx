import { NavLink } from 'react-router-dom';
import { ROUTES } from '../../lib/routes';

// Visual separators after these routes group the nav.
const SEP_AFTER = new Set(['/services', '/stubs/unused', '/verify']);

export function Sidebar({ collapsed }: Readonly<{ collapsed: boolean }>) {
  return (
    <nav style={{
      display: 'flex', flexDirection: 'column', gap: 2,
      padding: collapsed ? '8px 6px' : '8px 8px',
    }}>
      {ROUTES.map((item) => (
        <div key={item.to} style={{ display: 'contents' }}>
          <NavLink to={item.to} end={item.to === '/' || item.to === '/stubs'}
            title={collapsed ? item.label : undefined}
            style={({ isActive }) => ({
              position: 'relative',
              display: 'flex', alignItems: 'center',
              justifyContent: collapsed ? 'center' : 'flex-start',
              gap: 10,
              height: 36,
              padding: collapsed ? 0 : '0 10px',
              borderRadius: 'var(--radius)',
              fontSize: 13,
              fontWeight: isActive ? 600 : 500,
              color: isActive ? 'var(--accent-text)' : 'var(--text-secondary)',
              background: isActive ? 'var(--accent-bg)' : 'transparent',
              textDecoration: 'none',
              transition: 'background 0.12s, color 0.12s',
            })}
            className="sidebar-link">
            {({ isActive }) => (<>
              {isActive && !collapsed && (
                <span style={{ position: 'absolute', left: 0, top: 7, bottom: 7, width: 3, borderRadius: 3, background: 'var(--accent)' }} />
              )}
              <item.icon size={17} style={{ flexShrink: 0 }} />
              {!collapsed && <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.label}</span>}
            </>)}
          </NavLink>
          {SEP_AFTER.has(item.to) && <div style={{ height: 1, background: 'var(--border)', margin: collapsed ? '4px 6px' : '4px 8px', opacity: 0.7 }} />}
        </div>
      ))}
    </nav>
  );
}
