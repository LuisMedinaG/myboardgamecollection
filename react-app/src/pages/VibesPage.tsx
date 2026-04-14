import { useState } from 'react'
import { ALL_VIBES, GAMES } from '../data/games'
import GameListItem from '../components/GameListItem'

const VIBE_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  'Social':          { bg: '#dbeafe', text: '#1d4ed8', border: '#93c5fd' },
  'Competitive':     { bg: '#fee2e2', text: '#b91c1c', border: '#fca5a5' },
  'Chill':           { bg: '#ccfbf1', text: '#0f766e', border: '#5eead4' },
  'Family-Friendly': { bg: '#fef9c3', text: '#a16207', border: '#fde047' },
  'Tense':           { bg: '#ede9fe', text: '#6d28d9', border: '#c4b5fd' },
  'Co-op':           { bg: '#d1fae5', text: '#065f46', border: '#6ee7b7' },
  'Relaxing':        { bg: '#d1fae5', text: '#047857', border: '#34d399' },
  'Strategic':       { bg: '#e0e7ff', text: '#3730a3', border: '#a5b4fc' },
  'Cerebral':        { bg: '#f3e8ff', text: '#7e22ce', border: '#d8b4fe' },
  'Epic':            { bg: '#ffedd5', text: '#9a3412', border: '#fdba74' },
  'Fast':            { bg: '#fef3c7', text: '#92400e', border: '#fcd34d' },
  'Solo-Friendly':   { bg: '#f1f5f9', text: '#334155', border: '#94a3b8' },
  'Beautiful':       { bg: '#fce7f3', text: '#9d174d', border: '#f9a8d4' },
  'Party':           { bg: '#ecfccb', text: '#3f6212', border: '#bef264' },
}

function fallbackColor(vibe: string) {
  const colors = [
    { bg: '#dbeafe', text: '#1d4ed8', border: '#93c5fd' },
    { bg: '#f0fdf4', text: '#15803d', border: '#86efac' },
    { bg: '#fdf4ff', text: '#86198f', border: '#e879f9' },
  ]
  const idx = vibe.charCodeAt(0) % colors.length
  return colors[idx]
}

function vibePillColor(vibe: string) {
  return VIBE_COLORS[vibe] ?? fallbackColor(vibe)
}

export default function VibesPage() {
  const [selected, setSelected] = useState<string | null>(null)

  const filtered = selected
    ? GAMES.filter(g => g.vibes.includes(selected))
    : []

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      {/* Page header */}
      <div style={{ paddingTop: '0.25rem' }}>
        <h1 style={{
          fontFamily: 'var(--font-heading)',
          fontSize: '1.6rem',
          fontWeight: 700,
          color: 'var(--color-ink)',
          marginBottom: '0.1rem',
        }}>
          Browse by Vibe
        </h1>
        <p style={{ fontSize: '0.82rem', color: 'var(--color-muted)' }}>
          Pick a mood and find your next game
        </p>
      </div>

      {/* Vibe pill grid */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
        {ALL_VIBES.map(vibe => {
          const c = vibePillColor(vibe)
          const isSelected = vibe === selected
          return (
            <button
              key={vibe}
              onClick={() => setSelected(isSelected ? null : vibe)}
              className="pressable"
              style={{
                background: isSelected ? c.text : c.bg,
                color: isSelected ? 'white' : c.text,
                border: `1.5px solid ${isSelected ? c.text : c.border}`,
                borderRadius: '9999px',
                padding: '0.4rem 1rem',
                fontSize: '0.875rem',
                fontWeight: 600,
                fontFamily: 'var(--font-sans)',
                cursor: 'pointer',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '0.3rem',
                boxShadow: isSelected ? `0 2px 8px ${c.border}` : 'none',
              }}
            >
              {isSelected && <span style={{ fontSize: '0.75rem' }}>✓</span>}
              {vibe}
            </button>
          )
        })}
      </div>

      {/* Results */}
      {selected && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          <div style={{ fontSize: '0.8rem', color: 'var(--color-muted)' }}>
            {filtered.length} {filtered.length === 1 ? 'game' : 'games'} with "{selected}" vibe
          </div>
          {filtered.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--color-muted)' }}>
              <div style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>🎲</div>
              <div style={{ fontFamily: 'var(--font-heading)', fontSize: '1rem' }}>No games found</div>
            </div>
          ) : (
            filtered.map(g => <GameListItem key={g.id} game={g} />)
          )}
        </div>
      )}

      {/* Empty state */}
      {!selected && (
        <div style={{
          textAlign: 'center',
          padding: '3rem 1rem',
          color: 'var(--color-muted)',
        }}>
          <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem' }}>✦</div>
          <div style={{ fontFamily: 'var(--font-heading)', fontSize: '1.05rem', marginBottom: '0.3rem' }}>
            Choose a vibe above
          </div>
          <div style={{ fontSize: '0.82rem' }}>
            See which games match your mood
          </div>
        </div>
      )}
    </div>
  )
}
