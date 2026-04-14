import { useState, useEffect } from 'react'

interface Props {
  value: string
  onChange: (value: string) => void
}

export default function SearchInput({ value, onChange }: Props) {
  const [local, setLocal] = useState(value)

  useEffect(() => {
    setLocal(value)
  }, [value])

  useEffect(() => {
    const id = setTimeout(() => onChange(local), 300)
    return () => clearTimeout(id)
  }, [local]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <span style={{
        position: 'absolute',
        left: '0.75rem',
        top: '50%',
        transform: 'translateY(-50%)',
        color: 'var(--color-muted)',
        pointerEvents: 'none',
        fontSize: '0.9rem',
      }}>
        🔍
      </span>
      <input
        type="search"
        placeholder="Search games…"
        value={local}
        onChange={e => setLocal(e.target.value)}
        style={{
          width: '100%',
          border: '1px solid var(--color-edge)',
          borderRadius: '0.5rem',
          padding: '0.375rem 0.75rem 0.375rem 2.25rem',
          fontSize: '0.875rem',
          fontFamily: 'var(--font-sans)',
          background: 'var(--color-surface)',
          color: 'var(--color-ink)',
          outline: 'none',
        }}
        onFocus={e => (e.target.style.borderColor = 'var(--color-accent)')}
        onBlur={e => (e.target.style.borderColor = 'var(--color-edge)')}
      />
    </div>
  )
}
