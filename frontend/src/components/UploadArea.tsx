import { useRef, useState } from 'react'

interface Props {
  onUpload: (file: File) => void
  loading: boolean
}

export default function UploadArea({ onUpload, loading }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  function handleFiles(files: FileList | null) {
    if (!files || files.length === 0) return
    const file = files[0]
    if (!file.name.toLowerCase().endsWith('.pdf')) {
      alert('Please select a PDF file.')
      return
    }
    onUpload(file)
  }

  return (
    <div
      style={{
        border: `2px dashed ${dragging ? '#2563eb' : '#d1d5db'}`,
        borderRadius: 10,
        padding: '72px 40px',
        textAlign: 'center',
        cursor: loading ? 'wait' : 'pointer',
        background: dragging ? '#eff6ff' : '#f9fafb',
        transition: 'border-color 0.15s, background 0.15s',
        opacity: loading ? 0.65 : 1,
      }}
      onClick={() => !loading && inputRef.current?.click()}
      onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => {
        e.preventDefault()
        setDragging(false)
        if (!loading) handleFiles(e.dataTransfer.files)
      }}
    >
      <input
        ref={inputRef}
        type="file"
        accept=".pdf"
        style={{ display: 'none' }}
        onChange={(e) => handleFiles(e.target.files)}
      />
      <div style={{ fontSize: 36, marginBottom: 14, color: '#9ca3af' }}>[ PDF ]</div>
      {loading ? (
        <p style={{ margin: 0, fontWeight: 600, fontSize: 15, color: '#374151' }}>Uploading...</p>
      ) : (
        <>
          <p style={{ margin: '0 0 6px', fontWeight: 600, fontSize: 15, color: '#374151' }}>
            Drop a customer drawing here, or click to browse
          </p>
          <p style={{ margin: 0, color: '#9ca3af', fontSize: 13 }}>PDF files only</p>
        </>
      )}
    </div>
  )
}
