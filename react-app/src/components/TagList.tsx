interface Props {
  label: string
  tags: string[]
  variant: 'type' | 'category' | 'mechanic'
}

const CLASS_MAP = {
  type: 'tag tag-type',
  category: 'tag tag-category',
  mechanic: 'tag tag-mechanic',
}

export default function TagList({ label, tags, variant }: Props) {
  if (tags.length === 0) return null

  return (
    <div>
      <div style={{ fontSize: '0.72rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--color-muted)', marginBottom: '0.4rem' }}>
        {label}
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.375rem' }}>
        {tags.map(tag => (
          <span key={tag} className={CLASS_MAP[variant]}>{tag}</span>
        ))}
      </div>
    </div>
  )
}
