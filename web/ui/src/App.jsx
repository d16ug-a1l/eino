import { useState, useCallback } from 'react'
import Header from './components/Header'
import ChatArea from './components/ChatArea'
import ChatInput from './components/ChatInput'
import { useChat } from './hooks/useChat'

function getInitialTheme() {
  const saved = localStorage.getItem('eino-theme')
  if (saved) return saved
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export default function App() {
  const [theme, setTheme] = useState(getInitialTheme)
  const { messages, busy, connected, error, send, clear } = useChat()

  const toggleTheme = useCallback(() => {
    setTheme((prev) => {
      const next = prev === 'light' ? 'dark' : 'light'
      localStorage.setItem('eino-theme', next)
      return next
    })
  }, [])

  return (
    <div className="app" data-theme={theme}>
      <Header theme={theme} onToggleTheme={toggleTheme} connected={connected} error={error} />
      <ChatArea messages={messages} busy={busy} />
      <ChatInput onSend={send} onClear={clear} disabled={busy} />
    </div>
  )
}
