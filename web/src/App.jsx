import { useState, useEffect } from 'react'
import './App.css'

const API = 'http://localhost:8080'

function App() {
  const [entries, setEntries] = useState([])
  const [tags, setTags] = useState([])
  const [flatTags, setFlatTags] = useState([])
  const [content, setContent] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [selectedTag, setSelectedTag] = useState(null)
  const [breadcrumb, setBreadcrumb] = useState([])

  useEffect(() => {
    fetchEntries()
    fetchTags()
  }, [])

  async function fetchEntries(tagId = null) {
    try {
      const url = tagId
        ? `${API}/entries?tag=${tagId}`
        : `${API}/entries`
      const res = await fetch(url)
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
      setFlatTags(data.flat || [])
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
      fetchEntries(selectedTag?.id)
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
      fetchEntries(selectedTag?.id)
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

  function buildBreadcrumb(tagId) {
    const path = []
    let currentId = tagId
    while (currentId) {
      const tag = flatTags.find(t => t.id === currentId)
      if (tag) {
        path.unshift(tag)
        currentId = tag.parent_id
      } else {
        break
      }
    }
    return path
  }

  function selectTag(tag) {
    if (selectedTag?.id === tag.id) {
      // Deselect
      setSelectedTag(null)
      setBreadcrumb([])
      fetchEntries()
    } else {
      setSelectedTag(tag)
      setBreadcrumb(buildBreadcrumb(tag.id))
      fetchEntries(tag.id)
    }
    setSearch('')
  }

  function clearTagFilter() {
    setSelectedTag(null)
    setBreadcrumb([])
    fetchEntries()
  }

  function TagTree({ nodes, level = 0 }) {
    if (!nodes || nodes.length === 0) return null
    return (
      <ul className="tag-tree" style={{ paddingLeft: level > 0 ? '1rem' : 0 }}>
        {nodes.map(node => (
          <li key={node.id}>
            <button
              className={`tag-btn ${selectedTag?.id === node.id ? 'selected' : ''}`}
              onClick={() => selectTag(node)}
            >
              {node.name}
              {node.children && node.children.length > 0 && (
                <span className="child-count">({node.children.length})</span>
              )}
            </button>
            {node.children && <TagTree nodes={node.children} level={level + 1} />}
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
            <button type="button" onClick={() => { setSearch(''); fetchEntries(selectedTag?.id); }}>
              Clear
            </button>
          </form>
        </section>

        <div className="content-grid">
          <section className="entries-section">
            <div className="entries-header">
              <h2>Entries ({entries.length})</h2>
              {selectedTag && (
                <div className="filter-info">
                  <span className="filter-label">Filtered by:</span>
                  <nav className="breadcrumb">
                    <button className="breadcrumb-item root" onClick={clearTagFilter}>
                      All
                    </button>
                    {breadcrumb.map((tag, idx) => (
                      <span key={tag.id}>
                        <span className="breadcrumb-sep">›</span>
                        <button
                          className={`breadcrumb-item ${idx === breadcrumb.length - 1 ? 'current' : ''}`}
                          onClick={() => selectTag(tag)}
                        >
                          {tag.name}
                        </button>
                      </span>
                    ))}
                  </nav>
                  <button className="clear-filter" onClick={clearTagFilter}>×</button>
                </div>
              )}
            </div>
            <ul className="entries-list">
              {entries.map(entry => (
                <li key={entry.id} className="entry-card">
                  <p className="entry-content">{entry.content}</p>
                  {entry.tags && entry.tags.length > 0 && (
                    <div className="entry-tags">
                      {entry.tags.map(tag => (
                        <button
                          key={tag.id}
                          className={`tag ${selectedTag?.id === tag.id ? 'selected' : ''}`}
                          onClick={() => selectTag(tag)}
                        >
                          {tag.name}
                        </button>
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
            {selectedTag && (
              <button className="show-all-btn" onClick={clearTagFilter}>
                ← Show All Entries
              </button>
            )}
            <TagTree nodes={tags} />
            {tags.length === 0 && <p className="no-tags">No tags yet</p>}
          </aside>
        </div>
      </main>
    </div>
  )
}

export default App
