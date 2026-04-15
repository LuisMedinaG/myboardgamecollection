import { useState, useCallback, useEffect, useRef } from 'react'
import type { FilterState } from '../types/game'
import type { Game } from '../types/game'
import { api } from '../lib/api'
import FilterBar from '../components/FilterBar'
import ActiveFilters from '../components/ActiveFilters'
import GameListItem from '../components/GameListItem'
import GameCard from '../components/GameCard'

const EMPTY_FILTERS: FilterState = {
  search: '',
  category: '',
  players: '',
  playtime: '',
  weight: '',
}

function SkeletonRow() {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      gap: '0.75rem',
      padding: '0.75rem',
      background: 'var(--color-surface)',
      border: '1px solid var(--color-edge)',
      borderRadius: '0.875rem',
      boxShadow: 'var(--shadow-card)',
    }}>
      <div style={{ width: 56, height: 56, borderRadius: '0.5rem', background: 'var(--color-edge)', flexShrink: 0 }} />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
        <div style={{ height: '0.9rem', width: '60%', background: 'var(--color-edge)', borderRadius: '0.3rem' }} />
        <div style={{ height: '0.7rem', width: '40%', background: 'var(--color-edge)', borderRadius: '0.3rem' }} />
      </div>
    </div>
  )
}

export default function CollectionPage() {
  const [filters, setFilters] = useState<FilterState>(EMPTY_FILTERS)
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list')
  const [games, setGames] = useState<Game[]>([])
  const [total, setTotal] = useState(0)
  const [categories, setCategories] = useState<string[]>([])
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Only show skeleton on the very first load — after that, keep old games visible
  // during refetches (filter/search changes) to avoid layout flicker.
  const loading = !loaded

  const fetchGames = useCallback((f: FilterState) => {
    setError('')
    api.listGames({
      q:        f.search  || undefined,
      category: f.category || undefined,
      players:  f.players  || undefined,
      playtime: f.playtime || undefined,
      weight:   f.weight   || undefined,
      limit: 50,
      page: 1,
    }).then(res => {
      setGames(res.data)
      setTotal(res.total)
      if (res.categories.length > 0) setCategories(res.categories)
    }).catch(() => {
      setError('Failed to load games.')
    }).finally(() => {
      setLoaded(true)
    })
  }, [])

  // Initial load
  useEffect(() => { fetchGames(EMPTY_FILTERS) }, [fetchGames])

  const updateFilter = useCallback((key: keyof FilterState, value: string) => {
    setFilters(prev => {
      const next = { ...prev, [key]: value }
      if (key === 'search') {
        // Debounce search input
        if (searchTimerRef.current) clearTimeout(searchTimerRef.current)
        searchTimerRef.current = setTimeout(() => fetchGames(next), 300)
      } else {
        fetchGames(next)
      }
      return next
    })
  }, [fetchGames])

  const removeFilter = useCallback((key: keyof FilterState) => {
    setFilters(prev => {
      const next = { ...prev, [key]: '' }
      fetchGames(next)
      return next
    })
  }, [fetchGames])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
      {/* Page title */}
      <div style={{ paddingTop: '0.25rem' }}>
        <h1 style={{
          fontFamily: 'var(--font-heading)',
          fontSize: '1.6rem',
          fontWeight: 700,
          color: 'var(--color-ink)',
          marginBottom: '0.1rem',
        }}>
          Board Game Collection
        </h1>
        <p style={{ fontSize: '0.82rem', color: 'var(--color-muted)' }}>
          {loading ? 'Loading…' : `${total} games · find your next play`}
        </p>
      </div>

      {/* Filters */}
      <FilterBar filters={filters} categories={categories} onChange={updateFilter} />

      {/* Active filter chips */}
      <ActiveFilters filters={filters} onRemove={removeFilter} />

      {/* Result count + view toggle */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: '0.8rem', color: 'var(--color-muted)' }}>
          {loading ? '' : `${games.length} ${games.length === 1 ? 'game' : 'games'}`}
        </span>

        <div style={{ display: 'flex', gap: '0.25rem' }}>
          <button
            onClick={() => setViewMode('list')}
            aria-label="List view"
            style={{
              width: '32px',
              height: '32px',
              borderRadius: '0.4rem',
              border: '1px solid var(--color-edge)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              fontSize: '0.85rem',
              background: viewMode === 'list' ? 'var(--color-accent)' : 'var(--color-surface)',
              color: viewMode === 'list' ? 'white' : 'var(--color-muted)',
              transition: 'all 0.15s',
            }}
          >
            ☰
          </button>
          <button
            onClick={() => setViewMode('grid')}
            aria-label="Grid view"
            style={{
              width: '32px',
              height: '32px',
              borderRadius: '0.4rem',
              border: '1px solid var(--color-edge)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              fontSize: '0.85rem',
              background: viewMode === 'grid' ? 'var(--color-accent)' : 'var(--color-surface)',
              color: viewMode === 'grid' ? 'white' : 'var(--color-muted)',
              transition: 'all 0.15s',
            }}
          >
            ⊞
          </button>
        </div>
      </div>

      {/* Loading skeleton */}
      {loading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          {Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} />)}
        </div>
      )}

      {/* Error */}
      {!loading && error && (
        <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--color-danger, #b91c1c)' }}>
          {error}
        </div>
      )}

      {/* Game list / grid */}
      {!loading && !error && (
        games.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--color-muted)' }}>
            <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem' }}>🎲</div>
            <div style={{ fontFamily: 'var(--font-heading)', fontSize: '1.1rem', marginBottom: '0.4rem' }}>
              No games found
            </div>
            <div style={{ fontSize: '0.85rem' }}>
              Try adjusting your filters.
            </div>
          </div>
        ) : viewMode === 'list' ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
            {games.map(g => <GameListItem key={g.id} game={g} />)}
          </div>
        ) : (
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))',
            gap: '0.75rem',
          }}>
            {games.map(g => <GameCard key={g.id} game={g} />)}
          </div>
        )
      )}
    </div>
  )
}
