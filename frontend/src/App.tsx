import { useEffect, useState } from 'react'
import type { BOMRow, Document, DocumentStatus, Mapping } from './types/api'
import { analyzeDocument, checkAuth, exportCSVUrl, login, logout, saveBOM, saveMapping, uploadDocument } from './api/client'
import BomTable from './components/BomTable'
import LoginPage from './components/LoginPage'
import UploadArea from './components/UploadArea'
import WarningsPanel from './components/WarningsPanel'

export default function App() {
  const [authed, setAuthed] = useState<boolean | null>(null) // null = loading
  const [doc, setDoc] = useState<Document | null>(null)
  const [rows, setRows] = useState<BOMRow[]>([])
  const [uploading, setUploading] = useState(false)
  const [analyzing, setAnalyzing] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    checkAuth().then(ok => setAuthed(ok))
  }, [])

  async function handleLogin(username: string, password: string) {
    await login(username, password)
    setAuthed(true)
  }

  async function handleLogout() {
    await logout()
    setAuthed(false)
  }

  async function handleUpload(file: File) {
    setError(null)
    setUploading(true)
    try {
      const uploaded = await uploadDocument(file)
      setDoc(uploaded)
      setRows([])
      setSaved(false)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setUploading(false)
    }
  }

  async function handleAnalyze() {
    if (!doc) return
    setError(null)
    setAnalyzing(true)
    try {
      const result = await analyzeDocument(doc.id)
      setDoc(result)
      setRows(result.bomRows)
      setSaved(false)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setAnalyzing(false)
    }
  }

  async function handleSave() {
    if (!doc) return
    setError(null)
    try {
      await saveBOM(doc.id, rows)
      setSaved(true)
    } catch (e) {
      setError((e as Error).message)
    }
  }

  async function handleSaveMapping(
    mapping: Pick<Mapping, 'customerPartNumber' | 'internalPartNumber' | 'manufacturerPartNumber' | 'description' | 'source'>,
  ) {
    await saveMapping(mapping)
  }

  function handleRowsChange(next: BOMRow[]) {
    setRows(next)
    setSaved(false)
  }

  function handleReset() {
    setDoc(null)
    setRows([])
    setError(null)
    setSaved(false)
  }

  const hasResults = doc?.status === 'done' && rows.length > 0

  if (authed === null) return null // brief auth check on load
  if (!authed) return <LoginPage onLogin={handleLogin} />

  return (
    <div style={page}>
      <header style={header}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
          <div>
            <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>Drawing to BOM</h1>
            <p style={{ margin: '6px 0 0', color: '#6b7280', fontSize: 14 }}>
              Upload a customer drawing PDF to generate a draft Bill of Materials for review.
            </p>
          </div>
          <button style={ghostBtn} onClick={handleLogout}>Sign out</button>
        </div>
      </header>

      <main style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {error && (
          <div style={{ background: '#fee2e2', color: '#991b1b', padding: '12px 16px', borderRadius: 6, fontSize: 14, lineHeight: 1.5 }}>
            <strong>Error:</strong> {error}
          </div>
        )}

        {!doc ? (
          <UploadArea onUpload={handleUpload} loading={uploading} />
        ) : (
          <>
            {/* Document bar */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12, paddingBottom: 12, borderBottom: '1px solid #e5e7eb' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontWeight: 600, fontSize: 15 }}>{doc.filename}</span>
                <StatusBadge status={doc.status} />
              </div>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
                {(doc.status === 'uploaded' || doc.status === 'error') && (
                  <button style={primaryBtn} onClick={handleAnalyze} disabled={analyzing}>
                    {analyzing
                      ? <><span className="spinner" />Analyzing...</>
                      : 'Analyze Drawing'}
                  </button>
                )}
                {hasResults && (
                  <>
                    <button style={secondaryBtn} onClick={handleSave}>
                      {saved ? 'Saved' : 'Save Changes'}
                    </button>
                    <a href={exportCSVUrl(doc.id)} style={secondaryBtn} download>
                      Export CSV
                    </a>
                  </>
                )}
                <button style={ghostBtn} onClick={handleReset}>
                  Upload New
                </button>
              </div>
            </div>

            {/* Warnings */}
            <WarningsPanel warnings={doc.warnings} />

            {/* Results */}
            {hasResults ? (
              <BomTable rows={rows} onChange={handleRowsChange} onSaveMapping={handleSaveMapping} />
            ) : doc.status === 'uploaded' ? (
              <EmptyState>
                Drawing uploaded. Click <strong>Analyze Drawing</strong> to extract the BOM.
              </EmptyState>
            ) : doc.status === 'analyzing' ? (
              <EmptyState>Analyzing drawing...</EmptyState>
            ) : doc.status === 'error' ? (
              <EmptyState>Analysis failed. See error above.</EmptyState>
            ) : null}
          </>
        )}
      </main>
    </div>
  )
}

function StatusBadge({ status }: { status: DocumentStatus }) {
  const styles: Record<DocumentStatus, React.CSSProperties> = {
    uploaded:  { background: '#e0e7ff', color: '#3730a3' },
    analyzing: { background: '#fef3c7', color: '#92400e' },
    done:      { background: '#d1fae5', color: '#065f46' },
    error:     { background: '#fee2e2', color: '#991b1b' },
  }
  return (
    <span style={{ padding: '2px 10px', borderRadius: 12, fontSize: 12, fontWeight: 600, ...styles[status] }}>
      {status}
    </span>
  )
}

function EmptyState({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ padding: 48, textAlign: 'center', color: '#6b7280', border: '1px dashed #d1d5db', borderRadius: 8, fontSize: 14 }}>
      {children}
    </div>
  )
}

// Shared styles

const page: React.CSSProperties = {
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif',
  maxWidth: 1200,
  margin: '0 auto',
  padding: '0 24px 64px',
  color: '#111827',
}

const header: React.CSSProperties = {
  padding: '32px 0 20px',
  marginBottom: 24,
  borderBottom: '1px solid #e5e7eb',
}

const primaryBtn: React.CSSProperties = {
  padding: '8px 18px',
  background: '#2563eb',
  color: '#fff',
  border: 'none',
  borderRadius: 6,
  cursor: 'pointer',
  fontSize: 14,
  fontWeight: 600,
}

const secondaryBtn: React.CSSProperties = {
  padding: '8px 18px',
  background: '#f3f4f6',
  color: '#111827',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  cursor: 'pointer',
  fontSize: 14,
  textDecoration: 'none',
  display: 'inline-block',
}

const ghostBtn: React.CSSProperties = {
  padding: '8px 18px',
  background: 'transparent',
  color: '#6b7280',
  border: '1px solid #e5e7eb',
  borderRadius: 6,
  cursor: 'pointer',
  fontSize: 14,
}
