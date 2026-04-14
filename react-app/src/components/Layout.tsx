import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'

const glassStyle: React.CSSProperties = {
  backdropFilter: 'blur(20px) saturate(180%)',
  WebkitBackdropFilter: 'blur(20px) saturate(180%)',
  background: 'rgba(245,240,232,0.88)',
}

function TabBar() {
  const { pathname } = useLocation()
  const tabs = [
    { to: '/',       label: 'Collection', icon: '⊞' },
    { to: '/vibes',  label: 'Vibes',      icon: '✦' },
  ]

  return (
    <nav style={{
      position: 'fixed',
      bottom: 0,
      left: 0,
      right: 0,
      zIndex: 50,
      display: 'flex',
      borderTop: '0.5px solid rgba(212,197,169,0.6)',
      paddingBottom: 'env(safe-area-inset-bottom)',
      ...glassStyle,
    }}>
      {tabs.map(tab => {
        const active = tab.to === '/'
          ? pathname === '/' || pathname.startsWith('/games/')
          : pathname.startsWith(tab.to)
        return (
          <Link
            key={tab.to}
            to={tab.to}
            className="pressable"
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '3px',
              padding: '8px 0 4px',
              textDecoration: 'none',
              color: active ? 'var(--color-accent)' : 'var(--color-muted)',
            }}
          >
            <span style={{ fontSize: '1.25rem', lineHeight: 1 }}>{tab.icon}</span>
            <span style={{
              fontSize: '0.62rem',
              fontWeight: active ? 700 : 500,
              letterSpacing: '0.02em',
              textTransform: 'uppercase',
            }}>
              {tab.label}
            </span>
          </Link>
        )
      })}
    </nav>
  )
}

export default function Layout() {
  const location = useLocation()
  const navigate = useNavigate()
  const isDetail = location.pathname.startsWith('/games/')

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100dvh' }}>
      {/* Header */}
      <header style={{
        position: 'sticky',
        top: 0,
        zIndex: 50,
        borderBottom: '0.5px solid rgba(212,197,169,0.6)',
        paddingTop: 'env(safe-area-inset-top)',
        ...glassStyle,
      }}>
        <div style={{
          height: '52px',
          maxWidth: '640px',
          margin: '0 auto',
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          position: 'relative',
          padding: '0 1rem',
        }}>
          {/* Back button (detail pages) */}
          {isDetail && (
            <button
              onClick={() => navigate(-1)}
              className="pressable"
              style={{
                position: 'absolute',
                left: '1rem',
                background: 'none',
                border: 'none',
                padding: '0.25rem 0.5rem 0.25rem 0',
                display: 'flex',
                alignItems: 'center',
                gap: '3px',
                color: 'var(--color-accent)',
                fontSize: '1rem',
                fontWeight: 500,
                fontFamily: 'var(--font-sans)',
                cursor: 'pointer',
              }}
            >
              <span style={{ fontSize: '1.2rem', lineHeight: 1 }}>‹</span>
              <span style={{ fontSize: '0.9rem' }}>Collection</span>
            </button>
          )}

          {/* Center title */}
          <div style={{ flex: 1, textAlign: 'center' }}>
            {isDetail ? null : (
              <Link to="/" style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '0.4rem' }}>
                <span style={{ fontSize: '1.1rem' }}>🎲</span>
                <span style={{
                  fontFamily: 'var(--font-heading)',
                  fontWeight: 700,
                  fontSize: '1.05rem',
                  color: 'var(--color-ink)',
                  letterSpacing: '-0.01em',
                }}>
                  My Collection
                </span>
              </Link>
            )}
          </div>
        </div>
      </header>

      {/* Page content */}
      <main style={{
        flex: 1,
        maxWidth: '640px',
        margin: '0 auto',
        width: '100%',
        padding: '1rem 1rem 0',
        paddingBottom: 'calc(72px + env(safe-area-inset-bottom))',
      }}>
        <Outlet />
      </main>

      <TabBar />
    </div>
  )
}
