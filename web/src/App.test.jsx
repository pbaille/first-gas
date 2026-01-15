import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import App from './App'

// Mock fetch for API calls
beforeEach(() => {
  global.fetch = vi.fn(() =>
    Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ entries: [], tags: [], suggestions: [] }),
    })
  )
})

describe('App', () => {
  it('renders the Knowledge Base header', () => {
    render(<App />)
    expect(screen.getByText('Knowledge Base')).toBeInTheDocument()
  })

  it('renders the add entry textarea', () => {
    render(<App />)
    expect(screen.getByPlaceholderText(/what's on your mind/i)).toBeInTheDocument()
  })

  it('renders the search input', () => {
    render(<App />)
    expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument()
  })

  it('shows empty state messages', () => {
    render(<App />)
    expect(screen.getByText('No entries yet')).toBeInTheDocument()
    expect(screen.getByText('No tags yet')).toBeInTheDocument()
  })
})
