import { useState } from 'react'
import { ChevronDown, ChevronRight, Wrench } from 'lucide-react'

export default function ToolCallCard({ toolCall }) {
  const [open, setOpen] = useState(false)

  const name = toolCall.function?.name || 'unknown'
  let displayArgs = toolCall.function?.arguments || ''
  try {
    displayArgs = JSON.stringify(JSON.parse(displayArgs), null, 2)
  } catch {}

  return (
    <div className="tool-call-card">
      <div className="tool-call-header" onClick={() => setOpen(!open)}>
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        <Wrench size={13} />
        <span>调用工具: </span>
        <span className="tool-call-name">{name}</span>
      </div>
      {open && (
        <div className="tool-call-body">{displayArgs}</div>
      )}
    </div>
  )
}
