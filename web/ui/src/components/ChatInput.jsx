import { useState, useRef, useCallback } from 'react'
import { Send, Trash2 } from 'lucide-react'

export default function ChatInput({ onSend, onClear, disabled }) {
  const [text, setText] = useState('')
  const inputRef = useRef(null)

  const handleSend = useCallback(() => {
    const trimmed = text.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    setText('')
    inputRef.current?.focus()
  }, [text, disabled, onSend])

  const handleKeyDown = useCallback((e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }, [handleSend])

  const handleClear = useCallback(() => {
    onClear()
    inputRef.current?.focus()
  }, [onClear])

  return (
    <div className="chat-input-area">
      <div className="chat-input-wrapper">
        <textarea
          ref={inputRef}
          className="chat-input"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="输入消息，Enter 发送，Shift+Enter 换行"
          rows={1}
          disabled={disabled}
        />
        <button className="btn-icon" onClick={handleClear} title="清空对话" disabled={disabled}>
          <Trash2 size={16} />
        </button>
        <button className="btn-send" onClick={handleSend} disabled={!text.trim() || disabled} title="发送">
          <Send size={16} />
        </button>
      </div>
    </div>
  )
}
