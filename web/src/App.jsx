import { useState, useEffect } from 'react'
import './App.css'

const API = 'http://localhost:8080'

function App() {
  const [entries, setEntries] = useState([])
  const [tags, setTags] = useState([])
  const [suggestions, setSuggestions] = useState([])
  const [content, setContent] = useState('')
  const [search, setSearch] = useState('')
  const [selectedTag, setSelectedTag] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchTags()
    fetchSuggestions()
  }, [])

  useEffect(() => {
    fetchEntries()
  }, [search, selectedTag])

  async function fetchEntries() {
    try {
      const params = new URLSearchParams()
      if (search.trim()) params.set('q', search)
      if (selectedTag) params.set('tag', selectedTag)
      const url = params.toString() ? `${API}/entries?${params}` : `${API}/entries`
      const res = await fetch(url)
      const data = await res.json()
      setEntries(data.entries || [])
    } catch (err) {
      setError('Failed to fetch entries')
    }
  }

  function selectTag(tagName) {
    setSelectedTag(prev => prev === tagName ? null : tagName)
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

  async function fetchSuggestions() {
    try {
      const res = await fetch(`${API}/suggestions?limit=5`)
      const data = await res.json()
      setSuggestions(data.suggestions || [])
    } catch (err) {
      console.error('Failed to fetch suggestions')
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

  function TagTree({ nodes }) {
    if (!nodes || nodes.length === 0) return null
    return (
      <ul className="tag-tree">
        {nodes.map(node => (
          <li key={node.id}>
            <span
              className={`tag-name clickable ${selectedTag === node.name ? 'selected' : ''}`}
              onClick={() => selectTag(node.name)}
            >
              {node.name}
            </span>
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
          <div className="search-controls">
            <input
              type="text"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search entries..."
            />
            {(search || selectedTag) && (
              <button type="button" onClick={() => { setSearch(''); setSelectedTag(null); }}>
                Clear
              </button>
            )}
          </div>
          {selectedTag && (
            <div className="active-filter">
              Filtering by: <span className="tag selected">{selectedTag}</span>
              <button onClick={() => setSelectedTag(null)}>×</button>
            </div>
          )}
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
                        <span
                          key={tag.id}
                          className={`tag clickable ${selectedTag === tag.name ? 'selected' : ''}`}
                          onClick={() => selectTag(tag.name)}
                        >
                          {tag.name}
                        </span>
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

          <aside className="sidebar">
            <section className="tags-section">
              <h2>Tags</h2>
              <TagTree nodes={tags} />
              {tags.length === 0 && <p className="no-tags">No tags yet</p>}
            </section>

            <section className="suggestions-section">
              <h2>Suggestions</h2>
              {suggestions.length > 0 ? (
                <ul className="suggestions-list">
                  {suggestions.map(entry => (
                    <li key={entry.id} className="suggestion-card">
                      <p className="suggestion-content">{entry.content.slice(0, 100)}{entry.content.length > 100 ? '...' : ''}</p>
                      {entry.tags && entry.tags.length > 0 && (
                        <div className="entry-tags">
                          {entry.tags.map(tag => (
                            <span
                              key={tag.id}
                              className={`tag clickable ${selectedTag === tag.name ? 'selected' : ''}`}
                              onClick={() => selectTag(tag.name)}
                            >
                              {tag.name}
                            </span>
                          ))}
                        </div>
                      )}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="no-suggestions">No suggestions yet</p>
              )}
            </section>
          </aside>
        </div>
      </main>
    </div>
  )
}

export default App
