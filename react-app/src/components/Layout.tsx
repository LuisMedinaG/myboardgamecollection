import { Link, Outlet, useLocation } from 'react-router-dom'

export default function Layout() {
  const location = useLocation()
  const isDetail = location.pathname.startsWith('/games/')

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100dvh' }}>
      <header style={{
        position: 'sticky',
        top: 0,
        zIndex: 10,
        background: 'var(--color-accent)',
        boxShadow: 'var(--shadow-nav)',
        minHeight: '52px',
        display: 'flex',
        alignItems: 'center',
        padding: '0 1rem',
      }}>
        <div style={{ maxWidth: '640px', margin: '0 auto', width: '100%', display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          {isDetail && (
            <Link
              to="/"
              style={{
                color: 'white',
                fontSize: '1.1rem',
                lineHeight: 1,
                opacity: 0.9,
                display: 'flex',
                alignItems: 'center',
              }}
              aria-label="Back to collection"
            >
              ←
            </Link>
          )}
          <Link to="/" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', textDecoration: 'none' }}>
            <span style={{ fontSize: '1.25rem' }}>🎲</span>
            <span style={{
              fontFamily: 'var(--font-heading)',
              fontWeight: 700,
              fontSize: '1.1rem',
              color: 'white',
              letterSpacing: '-0.01em',
            }}>
              My Collection
            </span>
          </Link>
        </div>
      </header>

      <main style={{ flex: 1, maxWidth: '640px', margin: '0 auto', width: '100%', padding: '1rem 1rem 2rem' }}>
        <Outlet />
      </main>

      <footer style={{
        textAlign: 'center',
        padding: '1rem',
        fontSize: '0.75rem',
        color: 'var(--color-muted)',
        borderTop: '1px solid var(--color-edge)',
      }}>
        Board game data via BoardGameGeek
      </footer>
    </div>
  )
}
