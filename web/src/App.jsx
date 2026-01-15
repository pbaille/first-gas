import { useState, useEffect } from 'react'
import './App.css'

const API = 'http://localhost:8080'

function App() {
  const [entries, setEntries] = useState([])
  const [tags, setTags] = useState([])
  const [content, setContent] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchEntries()
    fetchTags()
  }, [])

  async function fetchEntries() {
    try {
      const res = await fetch(`${API}/entries`)
      const data = await res.json()
      setEntries(data.entries || [])
    } catch (err) {
      setError('Failed to fetch entries')
    }
  }

  async function fetchTags() {
    try {
      const res = await fetch(`${API}/tags`)
      const data = await res.json()
      setTags(data.tags || [])
    } catch (err) {
      console.error('Failed to fetch tags')
    }
  }

  async function addEntry(e) {
    e.preventDefault()
    if (!content.trim()) return

    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API}/entries`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content })
      })
      if (!res.ok) throw new Error('Failed to add entry')
      setContent('')
      fetchEntries()
      fetchTags()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function searchEntries(e) {
    e.preventDefault()
    if (!search.trim()) {
      fetchEntries()
      return
    }
    try {
      const res = await fetch(`${API}/search?q=${encodeURIComponent(search)}`)
      const data = await res.json()
      setEntries(data.entries || [])
    } catch (err) {
      setError('Search failed')
    }
  }

  function TagTree({ nodes }) {
    if (!nodes || nodes.length === 0) return null
    return (
      <ul className="tag-tree">
        {nodes.map(node => (
          <li key={node.id}>
            <span className="tag-name">{node.name}</span>
            {node.children && <TagTree nodes={node.children} />}
          </li>
        ))}
      </ul>
    )
  }

  return (
    <div className="app">
      <header>
        <h1>Knowledge Base</h1>
      </header>

      <main>
        <section className="add-section">
          <h2>Add Entry</h2>
          <form onSubmit={addEntry}>
            <textarea
              value={content}
              onChange={e => setContent(e.target.value)}
              placeholder="Enter your content..."
              rows={4}
            />
            <button type="submit" disabled={loading}>
              {loading ? 'Adding...' : 'Add Entry'}
            </button>
          </form>
          {error && <p className="error">{error}</p>}
        </section>

        <section className="search-section">
          <h2>Search</h2>
          <form onSubmit={searchEntries}>
            <input
              type="text"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search entries..."
            />
            <button type="submit">Search</button>
            <button type="button" onClick={() => { setSearch(''); fetchEntries(); }}>
              Clear
            </button>
          </form>
        </section>

        <div className="content-grid">
          <section className="entries-section">
            <h2>Entries ({entries.length})</h2>
            <ul className="entries-list">
              {entries.map(entry => (
                <li key={entry.id} className="entry-card">
                  <p className="entry-content">{entry.content}</p>
                  {entry.tags && entry.tags.length > 0 && (
                    <div className="entry-tags">
                      {entry.tags.map(tag => (
                        <span key={tag.id} className="tag">{tag.name}</span>
                      ))}
                    </div>
                  )}
                  <small className="entry-date">
                    {new Date(entry.created_at).toLocaleString()}
                  </small>
                </li>
              ))}
              {entries.length === 0 && <li className="no-entries">No entries yet</li>}
            </ul>
          </section>

          <aside className="tags-section">
            <h2>Tags</h2>
            <TagTree nodes={tags} />
            {tags.length === 0 && <p className="no-tags">No tags yet</p>}
          </aside>
        </div>
      </main>
    </div>
  )
}

export default App
