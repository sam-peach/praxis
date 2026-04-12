import { useRef, useState } from 'react'
import { colors, radius, shadow } from '../theme'

interface Props {
  onUpload: (files: File[]) => void
  loading:  boolean
  compact?: boolean
}

export default function UploadArea({ onUpload, loading, compact = false }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  function handleFiles(files: FileList | null) {
    if (!files || files.length === 0) return
    const pdfs = Array.from(files).filter(f => f.name.toLowerCase().endsWith('.pdf'))
    if (pdfs.length === 0) {
      alert('Please select PDF files.')
      return
    }
    onUpload(pdfs)
  }

  if (compact) {
    return (
      <div
        style={{
          display:      'flex',
          alignItems:   'center',
          gap:          10,
          padding:      '10px 14px',
          border:       `1.5px dashed ${dragging ? colors.brand : colors.border}`,
          borderRadius: radius.lg,
          background:   dragging ? colors.brandFaint : colors.surface,
          cursor:       loading ? 'wait' : 'pointer',
          transition:   'border-color 0.15s, background 0.15s',
          opacity:      loading ? 0.65 : 1,
        }}
        onClick={() => !loading && inputRef.current?.click()}
        onDragOver={e  => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={e => {
          e.preventDefault()
          setDragging(false)
          if (!loading) handleFiles(e.dataTransfer.files)
        }}
      >
        <input ref={inputRef} type="file" accept=".pdf" multiple style={{ display: 'none' }}
          onChange={e => handleFiles(e.target.files)} />
        <UploadIcon active={dragging} size={20} />
        <span style={{ fontSize: 13, color: colors.textMuted }}>
          {loading ? 'Uploading…' : 'Drop more PDFs or click to add'}
        </span>
      </div>
    )
  }

  return (
    <div
      style={{
        border:       `2px dashed ${dragging ? colors.brand : colors.border}`,
        borderRadius: radius.xl,
        padding:      '72px 40px',
        textAlign:    'center',
        cursor:       loading ? 'wait' : 'pointer',
        background:   dragging ? colors.brandFaint : colors.surface,
        transition:   'border-color 0.15s, background 0.15s',
        opacity:      loading ? 0.65 : 1,
        boxShadow:    shadow.sm,
      }}
      onClick={() => !loading && inputRef.current?.click()}
      onDragOver={e  => { e.preventDefault(); setDragging(true) }}
      onDragLeave={() => setDragging(false)}
      onDrop={e => {
        e.preventDefault()
        setDragging(false)
        if (!loading) handleFiles(e.dataTransfer.files)
      }}
    >
      <input ref={inputRef} type="file" accept=".pdf" multiple style={{ display: 'none' }}
        onChange={e => handleFiles(e.target.files)} />

      <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 18 }}>
        <UploadIcon active={dragging} size={52} />
      </div>

      {loading ? (
        <p style={{ margin: 0, fontWeight: 600, fontSize: 15, color: colors.text }}>
          Uploading…
        </p>
      ) : (
        <>
          <p style={{ margin: '0 0 6px', fontWeight: 600, fontSize: 15, color: colors.text }}>
            Drop customer drawings here, or click to browse
          </p>
          <p style={{ margin: 0, color: colors.textSubtle, fontSize: 13 }}>
            PDF files only · max 32 MB · multiple files supported
          </p>
        </>
      )}
    </div>
  )
}

function UploadIcon({ active = false, size = 52 }: { active?: boolean; size?: number }) {
  const c = active ? colors.brand : '#b8b3bf'
  const s = size / 52
  return (
    <svg width={size} height={size} viewBox="0 0 52 52" fill="none" aria-hidden="true">
      <circle cx="26" cy="26" r="22" stroke={c} strokeWidth={1.5 / s} opacity="0.2" />
      <path d="M26 34V20" stroke={c} strokeWidth={2 / s} strokeLinecap="round" />
      <path d="M19 27l7-8 7 8" stroke={c} strokeWidth={2 / s} strokeLinecap="round" strokeLinejoin="round" />
      <path d="M18 37h16" stroke={c} strokeWidth={1.5 / s} strokeLinecap="round" />
    </svg>
  )
}
