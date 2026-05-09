import { useState, useCallback, useRef } from 'react'

let idCounter = 0
function nextId() {
  return `msg-${++idCounter}`
}

function createMessage(overrides = {}) {
  return {
    id: nextId(),
    role: 'assistant',
    content: '',
    toolCalls: null,
    isStreaming: false,
    agentName: null,
    toolName: null,
    actionType: null,
    ...overrides,
  }
}

export function useChat() {
  const [messages, setMessages] = useState([])
  const [busy, setBusy] = useState(false)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState(null)
  const abortRef = useRef(null)

  const send = useCallback((query) => {
    // Abort any in-flight request
    if (abortRef.current) {
      abortRef.current.close()
      abortRef.current = null
    }

    setBusy(true)
    setConnected(false)
    setError(null)

    const userMsg = createMessage({ role: 'user', content: query })

    setMessages((prev) => [...prev, userMsg])

    // Track tool calls being accumulated from streaming
    const toolCallsByIndex = {}

    // Use a placeholder for the streaming assistant message
    let streamMsg = createMessage({ role: 'assistant', content: '', isStreaming: true })
    let streamMsgAdded = false

    const addStreamMsg = () => {
      if (!streamMsgAdded) {
        setMessages((prev) => [...prev, streamMsg])
        streamMsgAdded = true
      }
    }

    const es = new EventSource(`/chat?query=${encodeURIComponent(query)}`)

    abortRef.current = es

    es.onopen = () => {
      setConnected(true)
      setError(null)
    }

    es.onmessage = (e) => {
      let event
      try {
        event = JSON.parse(e.data)
      } catch {
        return
      }

      switch (event.type) {
        case 'stream_chunk': {
          addStreamMsg()
          streamMsg = { ...streamMsg, content: streamMsg.content + event.content }
          setMessages((prev) => {
            const updated = [...prev]
            const idx = updated.findIndex((m) => m.id === streamMsg.id)
            if (idx >= 0) updated[idx] = streamMsg
            return updated
          })
          break
        }

        case 'tool_result_chunk': {
          addStreamMsg()
          streamMsg = {
            ...streamMsg,
            content: streamMsg.content + event.content,
            role: 'tool',
            toolName: event.agent_name,
          }
          setMessages((prev) => {
            const updated = [...prev]
            const idx = updated.findIndex((m) => m.id === streamMsg.id)
            if (idx >= 0) updated[idx] = streamMsg
            return updated
          })
          break
        }

        case 'tool_calls': {
          addStreamMsg()
          streamMsg = {
            ...streamMsg,
            isStreaming: false,
            toolCalls: event.tool_calls || [],
          }
          setMessages((prev) => {
            const updated = [...prev]
            const idx = updated.findIndex((m) => m.id === streamMsg.id)
            if (idx >= 0) updated[idx] = streamMsg
            return updated
          })
          break
        }

        case 'message': {
          addStreamMsg()
          streamMsg = {
            ...streamMsg,
            content: event.content,
            isStreaming: false,
            agentName: event.agent_name,
            toolCalls: event.tool_calls || null,
          }
          setMessages((prev) => {
            const updated = [...prev]
            const idx = updated.findIndex((m) => m.id === streamMsg.id)
            if (idx >= 0) updated[idx] = streamMsg
            return updated
          })
          break
        }

        case 'tool_result': {
          addStreamMsg()
          streamMsg = {
            ...streamMsg,
            content: event.content,
            role: 'tool',
            toolName: event.agent_name,
            isStreaming: false,
          }
          setMessages((prev) => {
            const updated = [...prev]
            const idx = updated.findIndex((m) => m.id === streamMsg.id)
            if (idx >= 0) updated[idx] = streamMsg
            return updated
          })
          break
        }

        case 'action': {
          // Mark current stream as done
          if (streamMsgAdded) {
            streamMsg = { ...streamMsg, isStreaming: false }
            setMessages((prev) => {
              const updated = [...prev]
              const idx = updated.findIndex((m) => m.id === streamMsg.id)
              if (idx >= 0) updated[idx] = streamMsg
              return updated
            })
          }

          setMessages((prev) => [
            ...prev,
            createMessage({
              type: 'action',
              actionType: event.action_type,
              content: event.content,
              agentName: event.agent_name,
            }),
          ])
          break
        }

        case 'error': {
          setError(event.error || 'Unknown error')
          setMessages((prev) => [
            ...prev,
            createMessage({
              type: 'action',
              actionType: 'error',
              content: event.error || 'Unknown error',
              agentName: event.agent_name,
            }),
          ])
          break
        }

        default:
          break
      }
    }

    es.onerror = () => {
      // Mark stream as done
      if (streamMsgAdded) {
        streamMsg = { ...streamMsg, isStreaming: false }
        setMessages((prev) => {
          const updated = [...prev]
          const idx = updated.findIndex((m) => m.id === streamMsg.id)
          if (idx >= 0) updated[idx] = streamMsg
          return updated
        })
      }

      es.close()
      abortRef.current = null
      setBusy(false)
      setConnected(false)
    }
  }, [])

  const clear = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.close()
      abortRef.current = null
    }
    setMessages([])
    setBusy(false)
    setConnected(false)
    setError(null)
  }, [])

  return { messages, busy, connected, error, send, clear }
}
