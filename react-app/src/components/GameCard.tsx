import { Link } from 'react-router-dom'
import type { Game } from '../types/game'
import { playersStr, weightClass, weightLabel, imgFallback } from '../utils/gameFormatters'

interface Props {
  game: Game
}

export default function GameCard({ game }: Props) {
  return (
    <Link
      to={`/games/${game.id}`}
      style={{
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--color-surface)',
        border: '1px solid var(--color-edge)',
        borderRadius: '0.75rem',
        overflow: 'hidden',
        boxShadow: 'var(--shadow-card)',
        textDecoration: 'none',
        color: 'var(--color-ink)',
        transition: 'box-shadow 0.15s, transform 0.1s',
        WebkitTapHighlightColor: 'transparent',
      }}
      onMouseEnter={e => {
        e.currentTarget.style.boxShadow = '0 4px 16px rgba(44,32,8,0.15)'
        e.currentTarget.style.transform = 'scale(1.02)'
      }}
      onMouseLeave={e => {
        e.currentTarget.style.boxShadow = 'var(--shadow-card)'
        e.currentTarget.style.transform = ''
      }}
    >
      <div style={{ aspectRatio: '1', overflow: 'hidden' }}>
        <img
          src={game.thumbnail}
          alt={game.name}
          onError={e => { e.currentTarget.src = imgFallback(game.name) }}
          style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
        />
      </div>

      <div style={{ padding: '0.6rem 0.6rem 0.7rem', flex: 1, display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
        <div style={{
          fontFamily: 'var(--font-heading)',
          fontWeight: 600,
          fontSize: '0.85rem',
          lineHeight: 1.2,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          display: '-webkit-box',
          WebkitLineClamp: 2,
          WebkitBoxOrient: 'vertical',
        }}>
          {game.name}
        </div>

        <div style={{ fontSize: '0.72rem', color: 'var(--color-muted)' }}>
          {playersStr(game)}p · {game.playTime}m
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', marginTop: 'auto', paddingTop: '0.15rem' }}>
          <span className={weightClass(game.weight)}>{weightLabel(game.weight)}</span>
          <span style={{
            marginLeft: 'auto',
            fontSize: '0.72rem',
            fontWeight: 700,
            color: 'var(--color-rating)',
          }}>
            ★ {game.rating.toFixed(1)}
          </span>
        </div>
      </div>
    </Link>
  )
}
