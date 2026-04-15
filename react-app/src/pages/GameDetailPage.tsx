import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api, type GameDetail, type Collection } from '../lib/api'
import { playersStr, weightClass, weightLabel, imgFallback } from '../utils/gameFormatters'
import TagList from '../components/TagList'
import PlayerAidManager from '../components/PlayerAidManager'
import RulesUrlEditor from '../components/RulesUrlEditor'

const LANG_DEP = ['', 'No language', 'Some text', 'Moderate', 'Extensive', 'Unplayable']

const cardStyle: React.CSSProperties = {
  background: 'var(--color-surface)',
  border: '1px solid var(--color-edge)',
  borderRadius: '0.875rem',
  boxShadow: 'var(--shadow-card)',
  padding: '1rem',
  marginBottom: '0.75rem',
}

const sectionLabel: React.CSSProperties = {
  fontSize: '0.85rem',
  fontWeight: 700,
  marginBottom: '0.75rem',
  color: 'var(--color-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.07em',
}

export default function GameDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [game, setGame] = useState<GameDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [descExpanded, setDescExpanded] = useState(false)

  // Collections for vibes editing
  const [allCollections, setAllCollections] = useState<Collection[]>([])
  const [selectedVibeIds, setSelectedVibeIds] = useState<Set<number>>(new Set())
  const [editingVibes, setEditingVibes] = useState(false)
  const [savingVibes, setSavingVibes] = useState(false)

  // Delete
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError('')
    api.getGame(Number(id))
      .then(gameData => {
        setGame(gameData)
        setSelectedVibeIds(new Set(gameData.vibeCollectionIds))
        // Load collections in the background — failure doesn't block the page
        api.listCollections()
          .then(setAllCollections)
          .catch(() => {})
      })
      .catch(() => setError('Game not found.'))
      .finally(() => setLoading(false))
  }, [id])

  async function handleSaveVibes() {
    if (!game) return
    setSavingVibes(true)
    try {
      await api.setGameCollections(game.id, [...selectedVibeIds])
      const savedNames = allCollections.filter(c => selectedVibeIds.has(c.id)).map(c => c.name)
      setGame(prev => prev ? { ...prev, vibes: savedNames, vibeCollectionIds: [...selectedVibeIds] } : prev)
      setEditingVibes(false)
    } catch {
      // silently ignore — could add error state
    } finally {
      setSavingVibes(false)
    }
  }

  function toggleVibe(id: number) {
    setSelectedVibeIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function handleDelete() {
    if (!game) return
    setDeleting(true)
    try {
      await api.deleteGame(game.id)
      navigate('/')
    } catch {
      setDeleting(false)
      setConfirmDelete(false)
    }
  }

  if (loading) {
    return (
      <div style={{ paddingBottom: '0.5rem' }}>
        <div style={{ margin: '0 -1rem', height: '240px', background: 'var(--color-edge)' }} />
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem', marginTop: '1rem' }}>
          {[1, 2, 3].map(i => (
            <div key={i} style={{ height: '80px', background: 'var(--color-edge)', borderRadius: '0.875rem' }} />
          ))}
        </div>
      </div>
    )
  }

  if (error || !game) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 1rem', color: 'var(--color-muted)' }}>
        <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem' }}>🎲</div>
        <div style={{ fontFamily: 'var(--font-heading)', fontSize: '1.1rem', marginBottom: '0.75rem' }}>
          {error || 'Game not found.'}
        </div>
        <button
          onClick={() => navigate(-1)}
          className="btn btn-secondary pressable"
          style={{ padding: '0.6rem 1.25rem' }}
        >
          ‹ Back
        </button>
      </div>
    )
  }

  const bggUrl = `https://boardgamegeek.com/boardgame/${game.bggId}`

  return (
    <div style={{ paddingBottom: '0.5rem' }}>
      {/* Hero image */}
      <div style={{
        position: 'relative',
        margin: '0 -1rem',
        height: '240px',
        overflow: 'hidden',
        background: 'var(--color-edge)',
      }}>
        <img
          src={game.image || game.thumbnail}
          alt={game.name}
          onError={e => { e.currentTarget.src = imgFallback(game.name) }}
          style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
        />
        <div style={{
          position: 'absolute',
          inset: 0,
          background: 'linear-gradient(to bottom, transparent 35%, rgba(0,0,0,0.6))',
        }} />
        <div style={{ position: 'absolute', bottom: '1rem', left: '1rem', right: '1rem' }}>
          <h1 style={{
            fontSize: '1.6rem',
            fontWeight: 700,
            lineHeight: 1.15,
            color: 'white',
            textShadow: '0 1px 4px rgba(0,0,0,0.5)',
            marginBottom: '0.4rem',
          }}>
            {game.name}
          </h1>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: '0.4rem',
            fontSize: '0.8rem',
            color: 'rgba(255,255,255,0.85)',
          }}>
            {game.yearPublished > 0 && <span>{game.yearPublished}</span>}
            {game.rating > 0 && (
              <span style={{
                background: 'var(--color-rating)',
                color: 'white',
                borderRadius: '0.3rem',
                padding: '0.1rem 0.45rem',
                fontSize: '0.75rem',
                fontWeight: 700,
              }}>
                ★ {game.rating.toFixed(1)}
              </span>
            )}
            <span className={weightClass(game.weight)}>{weightLabel(game.weight)}</span>
            {game.languageDependence > 0 && (
              <span style={{
                background: 'rgba(0,0,0,0.4)',
                borderRadius: '0.3rem',
                padding: '0.1rem 0.45rem',
                fontSize: '0.75rem',
              }}>
                🗣 {LANG_DEP[game.languageDependence]}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Stats cards row */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, 1fr)',
        background: 'var(--color-surface)',
        border: '1px solid var(--color-edge)',
        borderRadius: '0.875rem',
        boxShadow: 'var(--shadow-card)',
        margin: '1rem 0',
        overflow: 'hidden',
      }}>
        {[
          { label: 'Players',    value: playersStr(game), sub: 'count' },
          { label: 'Playtime',   value: `${game.playTime}`, sub: 'minutes' },
          { label: 'Complexity', value: game.weight > 0 ? game.weight.toFixed(1) : '—', sub: '/ 5.0' },
        ].map((stat, i) => (
          <div
            key={stat.label}
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              padding: '1rem 0.5rem',
              borderRight: i < 2 ? '1px solid var(--color-edge)' : undefined,
            }}
          >
            <div style={{
              fontFamily: 'var(--font-heading)',
              fontSize: '1.5rem',
              fontWeight: 700,
              color: 'var(--color-ink)',
              lineHeight: 1,
            }}>
              {stat.value}
            </div>
            <div style={{
              fontSize: '0.62rem',
              fontWeight: 700,
              textTransform: 'uppercase',
              letterSpacing: '0.07em',
              color: 'var(--color-accent)',
              marginTop: '0.3rem',
            }}>
              {stat.label}
            </div>
            <div style={{ fontSize: '0.62rem', color: 'var(--color-muted)', opacity: 0.8 }}>
              {stat.sub}
            </div>
          </div>
        ))}
      </div>

      {/* Description */}
      {game.description && (
        <div style={cardStyle}>
          <h2 style={sectionLabel}>About</h2>
          <p style={{
            fontSize: '0.875rem',
            lineHeight: 1.65,
            color: 'var(--color-ink)',
            overflow: descExpanded ? undefined : 'hidden',
            display: descExpanded ? undefined : '-webkit-box',
            WebkitLineClamp: descExpanded ? undefined : 3,
            WebkitBoxOrient: descExpanded ? undefined : 'vertical',
          } as React.CSSProperties}>
            {game.description}
          </p>
          {game.description.length > 200 && (
            <button
              onClick={() => setDescExpanded(p => !p)}
              className="pressable"
              style={{
                background: 'none',
                border: 'none',
                padding: '0.35rem 0 0',
                fontSize: '0.82rem',
                color: 'var(--color-accent)',
                fontWeight: 600,
                cursor: 'pointer',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {descExpanded ? 'Show less ↑' : 'Read more ↓'}
            </button>
          )}
        </div>
      )}

      {/* Game tags */}
      {(game.types.length > 0 || game.categories.length > 0 || game.mechanics.length > 0) && (
        <div style={{ ...cardStyle, display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
          <TagList label="Type" tags={game.types} variant="type" />
          <TagList label="Categories" tags={game.categories} variant="category" />
          <TagList label="Mechanics" tags={game.mechanics} variant="mechanic" />
        </div>
      )}

      {/* Player aids — always shown (upload available) */}
      <PlayerAidManager gameId={game.id} initial={game.playerAids} />

      {/* Vibes / collections */}
      <div style={cardStyle}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
          <div style={{ ...sectionLabel, marginBottom: 0 }}>Vibes</div>
          {!editingVibes && (
            <button
              onClick={() => { setSelectedVibeIds(new Set(game.vibeCollectionIds)); setEditingVibes(true) }}
              className="pressable"
              style={{
                background: 'none',
                border: 'none',
                fontSize: '0.82rem',
                color: 'var(--color-accent)',
                fontWeight: 600,
                cursor: 'pointer',
                fontFamily: 'var(--font-sans)',
                padding: '0.2rem 0',
              }}
            >
              Edit
            </button>
          )}
        </div>

        {editingVibes ? (
          <>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.375rem', marginBottom: '0.75rem' }}>
              {allCollections.map(c => (
                <label
                  key={c.id}
                  style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.9rem', color: 'var(--color-ink)' }}
                >
                  <input
                    type="checkbox"
                    checked={selectedVibeIds.has(c.id)}
                    onChange={() => toggleVibe(c.id)}
                    style={{ width: '1rem', height: '1rem', accentColor: 'var(--color-accent)', cursor: 'pointer' }}
                  />
                  {c.name}
                </label>
              ))}
              {allCollections.length === 0 && (
                <div style={{ fontSize: '0.85rem', color: 'var(--color-muted)' }}>No collections yet.</div>
              )}
            </div>
            <div style={{ display: 'flex', gap: '0.5rem' }}>
              <button
                onClick={handleSaveVibes}
                disabled={savingVibes}
                className="btn btn-primary pressable"
                style={{ padding: '0.4rem 0.875rem', fontSize: '0.85rem' }}
              >
                {savingVibes ? 'Saving…' : 'Save'}
              </button>
              <button
                onClick={() => setEditingVibes(false)}
                className="btn btn-secondary pressable"
                style={{ padding: '0.4rem 0.875rem', fontSize: '0.85rem' }}
              >
                Cancel
              </button>
            </div>
          </>
        ) : (
          game.vibes.length > 0 ? (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.375rem' }}>
              {game.vibes.map(v => (
                <span key={v} className="vibe-pill">{v}</span>
              ))}
            </div>
          ) : (
            <div style={{ fontSize: '0.85rem', color: 'var(--color-muted)' }}>No vibes assigned.</div>
          )
        )}
      </div>

      {/* External links */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', marginBottom: '0.75rem' }}>
        <RulesUrlEditor gameId={game.id} initial={game.rulesUrl} />

        <a
          href={bggUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="pressable"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '0.75rem',
            background: 'var(--color-surface)',
            border: '1px solid var(--color-edge)',
            borderRadius: '0.875rem',
            padding: '0.875rem 1rem',
            boxShadow: 'var(--shadow-card)',
            color: 'var(--color-ink)',
          }}
        >
          <span style={{ fontSize: '1.25rem' }}>🎲</span>
          <span style={{ flex: 1, fontSize: '0.9rem', fontWeight: 600 }}>View on BoardGameGeek</span>
          <span style={{ color: 'var(--color-muted)', fontSize: '1rem' }}>↗</span>
        </a>
      </div>

      {/* Delete game */}
      <div style={{
        borderTop: '1px solid var(--color-edge)',
        paddingTop: '1rem',
        marginTop: '0.25rem',
      }}>
        {confirmDelete ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', flexWrap: 'wrap' }}>
            <span style={{ fontSize: '0.875rem', color: 'var(--color-ink)', fontWeight: 600 }}>
              Delete "{game.name}"?
            </span>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="pressable"
              style={{
                padding: '0.45rem 0.875rem',
                fontSize: '0.85rem',
                fontWeight: 600,
                borderRadius: '0.5rem',
                background: '#dc2626',
                color: 'white',
                border: 'none',
                cursor: deleting ? 'not-allowed' : 'pointer',
                fontFamily: 'var(--font-sans)',
                opacity: deleting ? 0.6 : 1,
              }}
            >
              {deleting ? 'Deleting…' : 'Yes, delete'}
            </button>
            <button
              onClick={() => setConfirmDelete(false)}
              className="btn btn-secondary pressable"
              style={{ padding: '0.45rem 0.875rem', fontSize: '0.85rem' }}
            >
              Cancel
            </button>
          </div>
        ) : (
          <button
            onClick={() => setConfirmDelete(true)}
            className="pressable"
            style={{
              background: 'none',
              border: 'none',
              padding: 0,
              fontSize: '0.85rem',
              color: '#dc2626',
              cursor: 'pointer',
              fontFamily: 'var(--font-sans)',
              fontWeight: 600,
            }}
          >
            Delete game
          </button>
        )}
      </div>
    </div>
  )
}
