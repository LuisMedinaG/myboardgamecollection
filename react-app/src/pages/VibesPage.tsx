import { useState, useEffect } from 'react'
import { api, type Collection } from '../lib/api'
import type { Game } from '../types/game'
import GameListItem from '../components/GameListItem'

const PALETTE = [
  { bg: '#dbeafe', text: '#1d4ed8', border: '#93c5fd' },
  { bg: '#fee2e2', text: '#b91c1c', border: '#fca5a5' },
  { bg: '#ccfbf1', text: '#0f766e', border: '#5eead4' },
  { bg: '#fef9c3', text: '#a16207', border: '#fde047' },
  { bg: '#ede9fe', text: '#6d28d9', border: '#c4b5fd' },
  { bg: '#d1fae5', text: '#065f46', border: '#6ee7b7' },
  { bg: '#e0e7ff', text: '#3730a3', border: '#a5b4fc' },
  { bg: '#f3e8ff', text: '#7e22ce', border: '#d8b4fe' },
  { bg: '#ffedd5', text: '#9a3412', border: '#fdba74' },
  { bg: '#fef3c7', text: '#92400e', border: '#fcd34d' },
  { bg: '#fce7f3', text: '#9d174d', border: '#f9a8d4' },
  { bg: '#ecfccb', text: '#3f6212', border: '#bef264' },
]

function pillColor(idx: number) {
  return PALETTE[idx % PALETTE.length]
}

export default function VibesPage() {
  const [collections, setCollections] = useState<Collection[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [games, setGames] = useState<Game[]>([])
  const [discoverTotal, setDiscoverTotal] = useState(0)
  const [loadingCollections, setLoadingCollections] = useState(true)
  const [loadingGames, setLoadingGames] = useState(false)

  useEffect(() => {
    api.listCollections()
      .then(data => setCollections(data))
      .catch(() => {})
      .finally(() => setLoadingCollections(false))
  }, [])

  function selectCollection(id: number) {
    if (id === selectedId) {
      setSelectedId(null)
      setGames([])
      return
    }
    setSelectedId(id)
    setLoadingGames(true)
    api.discover({ collection_id: id })
      .then(res => {
        setGames(res.data)
        setDiscoverTotal(res.total)
      })
      .catch(() => {
        setGames([])
        setDiscoverTotal(0)
      })
      .finally(() => setLoadingGames(false))
  }

  const selectedName = collections.find(c => c.id === selectedId)?.name ?? ''

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

      {/* Collection pills */}
      {loadingCollections ? (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} style={{ height: '36px', width: `${70 + (i * 17) % 50}px`, background: 'var(--color-edge)', borderRadius: '9999px' }} />
          ))}
        </div>
      ) : (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
          {collections.map((col, idx) => {
            const c = pillColor(idx)
            const isSelected = col.id === selectedId
            return (
              <button
                key={col.id}
                onClick={() => selectCollection(col.id)}
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
                {col.name}
                {col.gameCount > 0 && (
                  <span style={{
                    fontSize: '0.7rem',
                    background: isSelected ? 'rgba(255,255,255,0.25)' : c.border,
                    color: isSelected ? 'white' : c.text,
                    borderRadius: '9999px',
                    padding: '0.05rem 0.45rem',
                    fontWeight: 700,
                  }}>
                    {col.gameCount}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      )}

      {/* Results */}
      {selectedId !== null && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          {loadingGames ? (
            <>
              <div style={{ height: '0.8rem', width: '40%', background: 'var(--color-edge)', borderRadius: '0.3rem' }} />
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} style={{ height: '72px', background: 'var(--color-edge)', borderRadius: '0.875rem' }} />
              ))}
            </>
          ) : (
            <>
              <div style={{ fontSize: '0.8rem', color: 'var(--color-muted)' }}>
                {discoverTotal} {discoverTotal === 1 ? 'game' : 'games'} with "{selectedName}" vibe
              </div>
              {games.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--color-muted)' }}>
                  <div style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>🎲</div>
                  <div style={{ fontFamily: 'var(--font-heading)', fontSize: '1rem' }}>No games found</div>
                </div>
              ) : (
                games.map(g => <GameListItem key={g.id} game={g} />)
              )}
            </>
          )}
        </div>
      )}

      {/* Empty state */}
      {selectedId === null && !loadingCollections && (
        <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--color-muted)' }}>
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
