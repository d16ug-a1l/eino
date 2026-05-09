import { User, Bot, Wrench } from 'lucide-react'
import ToolCallCard from './ToolCallCard'

export default function MessageBubble({ message }) {
  const { role, content, toolCalls, isStreaming, agentName, toolName } = message

  const avatarIcon = role === 'user'
    ? <User size={16} />
    : role === 'tool'
      ? <Wrench size={16} />
      : <Bot size={16} />

  return (
    <div className={`message ${role}${isStreaming ? ' streaming' : ''}`}>
      <div className="message-avatar">{avatarIcon}</div>
      <div>
        <div className="message-content">
          {content}
          {isStreaming && content === '' && (
            <span style={{ color: 'var(--text-tertiary)' }}>Thinking...</span>
          )}
        </div>
        {toolCalls && toolCalls.length > 0 && (
          <div className="tool-calls">
            {toolCalls.map((tc, i) => (
              <ToolCallCard key={tc.id || i} toolCall={tc} />
            ))}
          </div>
        )}
        <div className="message-meta">
          {role === 'tool' && toolName ? toolName : agentName || ''}
        </div>
      </div>
    </div>
  )
}
