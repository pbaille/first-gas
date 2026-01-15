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
    if (e) e.preventDefault()
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
      fetchSuggestions()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function deleteEntry(id) {
    if (!window.confirm('Are you sure you want to delete this entry?')) {
      return
    }
    try {
      const res = await fetch(`${API}/entries/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Failed to delete entry')
      fetchEntries()
      fetchSuggestions()
    } catch (err) {
      setError(err.message)
    }
  }

  function handleKeyDown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      addEntry()
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
        <section className="add-section hero">
          <form onSubmit={addEntry}>
            <textarea
              className="hero-input"
              value={content}
              onChange={e => setContent(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="What's on your mind? (⌘+Enter to save)"
              rows={3}
              autoFocus
            />
            <button type="submit" className="subtle-btn" disabled={loading}>
              {loading ? '...' : 'Save'}
            </button>
          </form>
          {error && <p className="error">{error}</p>}
        </section>

        <section className="search-section">
          <input
            type="text"
            className="search-input"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search..."
          />
          {selectedTag && (
            <div className="active-filter">
              <span className="tag selected">{selectedTag}</span>
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
                  <div className="entry-footer">
                    <small className="entry-date">
                      {new Date(entry.created_at).toLocaleString()}
                    </small>
                    <button
                      className="delete-btn"
                      onClick={() => deleteEntry(entry.id)}
                    >
                      Delete
                    </button>
                  </div>
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
