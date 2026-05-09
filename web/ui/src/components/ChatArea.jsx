import { useRef, useEffect } from 'react'
import { MessageSquare } from 'lucide-react'
import MessageBubble from './MessageBubble'
import ActionBanner from './ActionBanner'

export default function ChatArea({ messages, busy }) {
  const bottomRef = useRef(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, busy])

  if (messages.length === 0 && !busy) {
    return (
      <div className="chat-area">
        <div className="chat-empty">
          <div className="chat-empty-icon">
            <MessageSquare size={28} />
          </div>
          <h2>开始对话</h2>
          <p>在下方输入消息，与 AI Agent 交流</p>
        </div>
      </div>
    )
  }

  return (
    <div className="chat-area">
      {messages.map((msg) => {
        if (msg.type === 'action') {
          return <ActionBanner key={msg.id} message={msg} />
        }
        return <MessageBubble key={msg.id} message={msg} />
      })}
      <div ref={bottomRef} />
    </div>
  )
}
