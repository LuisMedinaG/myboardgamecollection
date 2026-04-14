import { useState, useCallback } from 'react'
import type { FilterState } from '../types/game'
import { GAMES, ALL_CATEGORIES } from '../data/games'
import { useFilteredGames } from '../hooks/useFilteredGames'
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

export default function CollectionPage() {
  const [filters, setFilters] = useState<FilterState>(EMPTY_FILTERS)
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list')

  const updateFilter = useCallback((key: keyof FilterState, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }))
  }, [])

  const removeFilter = useCallback((key: keyof FilterState) => {
    setFilters(prev => ({ ...prev, [key]: '' }))
  }, [])

  const filteredGames = useFilteredGames(GAMES, filters)

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
          {GAMES.length} games · find your next play
        </p>
      </div>

      {/* Filters */}
      <FilterBar filters={filters} categories={ALL_CATEGORIES} onChange={updateFilter} />

      {/* Active filter chips */}
      <ActiveFilters filters={filters} onRemove={removeFilter} />

      {/* Result count + view toggle */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: '0.8rem', color: 'var(--color-muted)' }}>
          {filteredGames.length} {filteredGames.length === 1 ? 'game' : 'games'}
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

      {/* Game list / grid */}
      {filteredGames.length === 0 ? (
        <div style={{
          textAlign: 'center',
          padding: '3rem 1rem',
          color: 'var(--color-muted)',
        }}>
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
          {filteredGames.map(g => <GameListItem key={g.id} game={g} />)}
        </div>
      ) : (
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))',
          gap: '0.75rem',
        }}>
          {filteredGames.map(g => <GameCard key={g.id} game={g} />)}
        </div>
      )}
    </div>
  )
}
