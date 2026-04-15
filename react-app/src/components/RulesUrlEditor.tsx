import { useState } from 'react'
import { api } from '../lib/api'

const DRIVE_RE = /^https:\/\/(drive|docs)\.google\.com\//

interface Props {
  gameId: number
  initial: string
}

const linkRowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: '0.75rem',
  flex: 1,
  background: 'var(--color-surface)',
  border: '1px solid var(--color-edge)',
  borderRadius: '0.875rem',
  padding: '0.875rem 1rem',
  boxShadow: 'var(--shadow-card)',
  color: 'var(--color-ink)',
  textDecoration: 'none',
}

export default function RulesUrlEditor({ gameId, initial }: Props) {
  const [url, setUrl] = useState(initial)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(initial)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSave() {
    const trimmed = draft.trim()
    if (trimmed && !DRIVE_RE.test(trimmed)) {
      setError('Must be a Google Drive or Docs URL')
      return
    }
    setSaving(true)
    setError('')
    try {
      await api.updateRulesUrl(gameId, trimmed)
      setUrl(trimmed)
      setEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  function handleCancel() {
    setDraft(url)
    setError('')
    setEditing(false)
  }

  if (editing) {
    return (
      <div style={{
        background: 'var(--color-surface)',
        border: '1px solid var(--color-edge)',
        borderRadius: '0.875rem',
        boxShadow: 'var(--shadow-card)',
        padding: '0.875rem 1rem',
      }}>
        <div style={{
          fontSize: '0.75rem',
          fontWeight: 700,
          textTransform: 'uppercase',
          letterSpacing: '0.07em',
          color: 'var(--color-muted)',
          marginBottom: '0.5rem',
        }}>
          Rulebook URL
        </div>
        <input
          type="url"
          value={draft}
          onChange={e => { setDraft(e.target.value); setError('') }}
          placeholder="https://drive.google.com/…"
          autoFocus
          style={{
            width: '100%',
            padding: '0.5rem 0.75rem',
            fontSize: '0.875rem',
            border: '1px solid var(--color-edge)',
            borderRadius: '0.5rem',
            background: 'var(--color-bg)',
            color: 'var(--color-ink)',
            fontFamily: 'var(--font-sans)',
            boxSizing: 'border-box',
          }}
        />
        {error && (
          <div style={{ fontSize: '0.8rem', color: '#dc2626', marginTop: '0.25rem' }}>
            {error}
          </div>
        )}
        <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem' }}>
          <button
            onClick={handleSave}
            disabled={saving}
            className="btn btn-primary pressable"
            style={{ padding: '0.4rem 0.875rem', fontSize: '0.85rem' }}
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button
            onClick={handleCancel}
            className="btn btn-secondary pressable"
            style={{ padding: '0.4rem 0.875rem', fontSize: '0.85rem' }}
          >
            Cancel
          </button>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
      {url ? (
        <a href={url} target="_blank" rel="noopener noreferrer" className="pressable" style={linkRowStyle}>
          <span style={{ fontSize: '1.25rem' }}>📖</span>
          <span style={{ flex: 1, fontSize: '0.9rem', fontWeight: 600 }}>Rulebook</span>
          <span style={{ color: 'var(--color-muted)', fontSize: '1rem' }}>↗</span>
        </a>
      ) : (
        <div style={{ ...linkRowStyle, color: 'var(--color-muted)' }}>
          <span style={{ fontSize: '1.25rem' }}>📖</span>
          <span style={{ flex: 1, fontSize: '0.9rem' }}>No rulebook link</span>
        </div>
      )}
      <button
        onClick={() => { setDraft(url); setEditing(true) }}
        className="pressable"
        title="Edit rulebook URL"
        style={{
          background: 'var(--color-surface)',
          border: '1px solid var(--color-edge)',
          borderRadius: '0.875rem',
          padding: '0.875rem',
          boxShadow: 'var(--shadow-card)',
          color: 'var(--color-muted)',
          cursor: 'pointer',
          fontSize: '1rem',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        ✏️
      </button>
    </div>
  )
}
