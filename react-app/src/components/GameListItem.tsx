import { Link } from 'react-router-dom'
import type { Game } from '../types/game'
import { playersStr, weightClass, weightLabel, imgFallback } from '../utils/gameFormatters'

interface Props {
  game: Game
}

export default function GameListItem({ game }: Props) {
  return (
    <Link
      to={`/games/${game.id}`}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '0.75rem',
        background: 'var(--color-surface)',
        border: '1px solid var(--color-edge)',
        borderRadius: '0.75rem',
        padding: '0.75rem 1rem',
        textDecoration: 'none',
        color: 'var(--color-ink)',
        boxShadow: 'var(--shadow-card)',
        transition: 'transform 0.1s',
        WebkitTapHighlightColor: 'transparent',
      }}
      onMouseEnter={e => (e.currentTarget.style.transform = 'scale(1.005)')}
      onMouseLeave={e => (e.currentTarget.style.transform = '')}
    >
      <img
        src={game.thumbnail}
        alt={game.name}
        onError={e => { e.currentTarget.src = imgFallback(game.name) }}
        style={{
          width: '56px',
          height: '56px',
          borderRadius: '0.5rem',
          objectFit: 'cover',
          flexShrink: 0,
        }}
      />

      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontWeight: 600, fontSize: '0.95rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {game.name}
        </div>
        <div style={{ fontSize: '0.78rem', color: 'var(--color-muted)', marginTop: '0.15rem' }}>
          {playersStr(game)} players · {game.playTime} min
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.25rem', marginTop: '0.35rem', alignItems: 'center' }}>
          <span className={weightClass(game.weight)}>{weightLabel(game.weight)}</span>
          {game.vibes.map(v => (
            <span key={v} className="vibe-pill">{v}</span>
          ))}
        </div>
      </div>

      <div style={{ fontSize: '1.2rem', color: 'var(--color-muted)', flexShrink: 0 }}>›</div>
    </Link>
  )
}
