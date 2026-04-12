interface Props {
  warnings: string[]
}

export default function WarningsPanel({ warnings }: Props) {
  if (!warnings || warnings.length === 0) return null
  return (
    <div
      style={{
        background: '#fffbeb',
        border: '1px solid #fbbf24',
        borderRadius: 6,
        padding: '12px 16px',
      }}
    >
      <strong style={{ display: 'block', marginBottom: 8, color: '#92400e', fontSize: 14 }}>
        Warnings &amp; Ambiguities
      </strong>
      <ul style={{ margin: 0, paddingLeft: 20 }}>
        {warnings.map((w, i) => (
          <li key={i} style={{ fontSize: 13, color: '#78350f', marginBottom: 4 }}>
            {w}
          </li>
        ))}
      </ul>
    </div>
  )
}
