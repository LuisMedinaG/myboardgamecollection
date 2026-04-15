import { useState, useEffect } from 'react'
import { api, type PlayerAid } from '../lib/api'

interface Props {
  gameId: number
  initial: PlayerAid[]
}

function navBtnStyle(side: 'left' | 'right'): React.CSSProperties {
  return {
    position: 'absolute',
    [side]: '1rem',
    top: '50%',
    transform: 'translateY(-50%)',
    background: 'rgba(255,255,255,0.15)',
    border: 'none',
    borderRadius: '50%',
    width: '2.5rem',
    height: '2.5rem',
    color: 'white',
    fontSize: '1.5rem',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    lineHeight: 1,
  }
}

export default function PlayerAidManager({ gameId, initial }: Props) {
  const [aids, setAids] = useState<PlayerAid[]>(initial)
  const [lightbox, setLightbox] = useState<number | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadErr, setUploadErr] = useState('')
  const [labelInput, setLabelInput] = useState('')

  useEffect(() => {
    if (lightbox === null) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setLightbox(null)
      else if (e.key === 'ArrowLeft') setLightbox(prev => (prev !== null && prev > 0 ? prev - 1 : prev))
      else if (e.key === 'ArrowRight') setLightbox(prev => (prev !== null && prev < aids.length - 1 ? prev + 1 : prev))
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [lightbox, aids.length])

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    setUploadErr('')
    try {
      const label = labelInput.trim() || file.name.replace(/\.[^.]+$/, '')
      const aid = await api.uploadPlayerAid(gameId, file, label)
      setAids(prev => [...prev, aid])
      setLabelInput('')
      e.target.value = ''
    } catch (err) {
      setUploadErr(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(aid: PlayerAid) {
    if (!confirm(`Delete "${aid.label}"?`)) return
    try {
      await api.deletePlayerAid(gameId, aid.id)
      setAids(prev => prev.filter(a => a.id !== aid.id))
      setLightbox(null)
    } catch {
      // ignore — could add toast
    }
  }

  const cur = lightbox !== null ? aids[lightbox] : null

  return (
    <>
      <div style={{
        background: 'var(--color-surface)',
        border: '1px solid var(--color-edge)',
        borderRadius: '0.875rem',
        boxShadow: 'var(--shadow-card)',
        padding: '1rem',
        marginBottom: '0.75rem',
      }}>
        <h2 style={{
          fontSize: '0.85rem',
          fontWeight: 700,
          marginBottom: aids.length > 0 ? '0.75rem' : '0.5rem',
          color: 'var(--color-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.07em',
        }}>
          Player Aids
        </h2>

        {aids.length > 0 && (
          <div style={{
            display: 'flex',
            gap: '0.75rem',
            overflowX: 'auto',
            paddingBottom: '0.5rem',
            marginBottom: '0.75rem',
          }}>
            {aids.map((aid, i) => (
              <div key={aid.id} style={{ flexShrink: 0, position: 'relative' }}>
                <button
                  onClick={() => setLightbox(i)}
                  className="pressable"
                  style={{ background: 'none', border: 'none', padding: 0, cursor: 'pointer', display: 'block' }}
                >
                  <img
                    src={`/uploads/${aid.filename}`}
                    alt={aid.label}
                    style={{
                      width: '120px',
                      height: '90px',
                      objectFit: 'cover',
                      borderRadius: '0.5rem',
                      border: '1px solid var(--color-edge)',
                      display: 'block',
                    }}
                  />
                  <div style={{
                    fontSize: '0.7rem',
                    color: 'var(--color-muted)',
                    marginTop: '0.25rem',
                    textAlign: 'center',
                    maxWidth: '120px',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}>
                    {aid.label}
                  </div>
                </button>
                <button
                  onClick={() => handleDelete(aid)}
                  title="Delete player aid"
                  style={{
                    position: 'absolute',
                    top: '4px',
                    right: '4px',
                    background: 'rgba(0,0,0,0.6)',
                    border: 'none',
                    borderRadius: '50%',
                    width: '20px',
                    height: '20px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer',
                    color: 'white',
                    fontSize: '0.65rem',
                    lineHeight: 1,
                    padding: 0,
                  }}
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        )}

        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap' }}>
          <input
            type="text"
            placeholder="Label (optional)"
            value={labelInput}
            onChange={e => setLabelInput(e.target.value)}
            style={{
              flex: 1,
              minWidth: '120px',
              padding: '0.45rem 0.75rem',
              fontSize: '0.85rem',
              border: '1px solid var(--color-edge)',
              borderRadius: '0.5rem',
              background: 'var(--color-bg)',
              color: 'var(--color-ink)',
              fontFamily: 'var(--font-sans)',
            }}
          />
          <label
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '0.35rem',
              padding: '0.45rem 0.875rem',
              fontSize: '0.85rem',
              fontWeight: 600,
              borderRadius: '0.5rem',
              background: uploading ? 'var(--color-edge)' : 'var(--color-accent)',
              color: uploading ? 'var(--color-muted)' : 'white',
              cursor: uploading ? 'not-allowed' : 'pointer',
              border: 'none',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {uploading ? 'Uploading…' : '+ Upload'}
            <input
              type="file"
              accept="image/png,image/jpeg,image/gif,image/webp"
              onChange={handleUpload}
              disabled={uploading}
              hidden
            />
          </label>
        </div>

        {uploadErr && (
          <div style={{ fontSize: '0.8rem', color: '#dc2626', marginTop: '0.35rem' }}>
            {uploadErr}
          </div>
        )}
      </div>

      {/* Lightbox */}
      {cur && (
        <div
          onClick={() => setLightbox(null)}
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.88)',
            zIndex: 1000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <img
            src={`/uploads/${cur.filename}`}
            alt={cur.label}
            onClick={e => e.stopPropagation()}
            style={{
              maxWidth: '90vw',
              maxHeight: '85vh',
              objectFit: 'contain',
              borderRadius: '0.5rem',
            }}
          />

          {lightbox! > 0 && (
            <button
              onClick={e => { e.stopPropagation(); setLightbox(prev => prev! - 1) }}
              className="pressable"
              style={navBtnStyle('left')}
            >
              ‹
            </button>
          )}

          {lightbox! < aids.length - 1 && (
            <button
              onClick={e => { e.stopPropagation(); setLightbox(prev => prev! + 1) }}
              className="pressable"
              style={navBtnStyle('right')}
            >
              ›
            </button>
          )}

          <button
            onClick={() => setLightbox(null)}
            className="pressable"
            style={{
              position: 'absolute',
              top: '1rem',
              right: '1rem',
              background: 'rgba(255,255,255,0.15)',
              border: 'none',
              borderRadius: '50%',
              width: '2rem',
              height: '2rem',
              color: 'white',
              fontSize: '1.1rem',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            ✕
          </button>

          {cur.label && (
            <div style={{
              position: 'absolute',
              bottom: '1rem',
              left: '50%',
              transform: 'translateX(-50%)',
              background: 'rgba(0,0,0,0.5)',
              color: 'white',
              fontSize: '0.85rem',
              padding: '0.25rem 0.75rem',
              borderRadius: '0.375rem',
              whiteSpace: 'nowrap',
            }}>
              {cur.label}
            </div>
          )}
        </div>
      )}
    </>
  )
}
