/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package web

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// processEvents drains the agent event iterator and sends serialized events to the writer.
func processEvents(ctx context.Context, w EventWriter, iter *adk.AsyncIterator[*adk.AgentEvent], checkpointID string) error {
	for {
		event, ok := iter.Next()
		if !ok {
			return nil
		}
		if err := convertAndSend(ctx, w, event, checkpointID); err != nil {
			return err
		}
	}
}

func convertAndSend(ctx context.Context, w EventWriter, event *adk.AgentEvent, checkpointID string) error {
	if event.Err != nil {
		return w.WriteEvent(ctx, &Event{
			Type:      EventTypeError,
			AgentName: event.AgentName,
			RunPath:   formatRunPath(event.RunPath),
			Error:     event.Err.Error(),
		})
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		if err := handleMessageOutput(ctx, w, event); err != nil {
			return err
		}
	}

	if event.Action != nil {
		if err := handleAction(ctx, w, event, checkpointID); err != nil {
			return err
		}
	}

	return nil
}

func handleMessageOutput(ctx context.Context, w EventWriter, event *adk.AgentEvent) error {
	mo := event.Output.MessageOutput

	if mo.Message != nil {
		return handleRegularMessage(ctx, w, event, mo)
	}
	if mo.MessageStream != nil {
		return handleStreamingMessage(ctx, w, event, mo.MessageStream)
	}
	return nil
}

func handleRegularMessage(ctx context.Context, w EventWriter, event *adk.AgentEvent, mo *adk.MessageVariant) error {
	eventType := EventTypeMessage
	if mo.Role == schema.Tool {
		eventType = EventTypeToolResult
	}
	evt := &Event{
		Type:      eventType,
		AgentName: event.AgentName,
		RunPath:   formatRunPath(event.RunPath),
		Content:   mo.Message.Content,
	}
	if len(mo.Message.ToolCalls) > 0 {
		evt.ToolCalls = mo.Message.ToolCalls
	}
	return w.WriteEvent(ctx, evt)
}

func handleStreamingMessage(ctx context.Context, w EventWriter, event *adk.AgentEvent, stream *schema.StreamReader[*schema.Message]) error {
	toolCallsByIndex := make(map[int][]*schema.Message)

	defer stream.Close()
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return w.WriteEvent(ctx, &Event{
				Type:      EventTypeError,
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Error:     fmt.Sprintf("stream error: %v", err),
			})
		}

		if chunk.Content != "" {
			chunkType := EventTypeStreamChunk
			if chunk.Role == schema.Tool {
				chunkType = EventTypeToolResultChunk
			}
			if err := w.WriteEvent(ctx, &Event{
				Type:      chunkType,
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Content:   chunk.Content,
			}); err != nil {
				return err
			}
		}

		for _, tc := range chunk.ToolCalls {
			if tc.Index != nil {
				toolCallsByIndex[*tc.Index] = append(toolCallsByIndex[*tc.Index], &schema.Message{
					Role:      chunk.Role,
					ToolCalls: []schema.ToolCall{tc},
				})
			}
		}
	}

	for _, msgs := range toolCallsByIndex {
		merged, err := schema.ConcatMessages(msgs)
		if err != nil {
			return w.WriteEvent(ctx, &Event{
				Type:      EventTypeError,
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Error:     fmt.Sprintf("tool call merge error: %v", err),
			})
		}
		if len(merged.ToolCalls) > 0 {
			if err := w.WriteEvent(ctx, &Event{
				Type:      EventTypeToolCalls,
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				ToolCalls: merged.ToolCalls,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func handleAction(ctx context.Context, w EventWriter, event *adk.AgentEvent, checkpointID string) error {
	action := event.Action

	if action.TransferToAgent != nil {
		return w.WriteEvent(ctx, &Event{
			Type:       EventTypeAction,
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: ActionTypeTransfer,
			Content:    action.TransferToAgent.DestAgentName,
		})
	}

	if action.Interrupted != nil {
		for _, ic := range action.Interrupted.InterruptContexts {
			content := fmt.Sprintf("%v", ic.Info)
			if stringer, ok := ic.Info.(fmt.Stringer); ok {
				content = stringer.String()
			}
			if err := w.WriteEvent(ctx, &Event{
				Type:          EventTypeAction,
				AgentName:     event.AgentName,
				RunPath:       formatRunPath(event.RunPath),
				ActionType:    ActionTypeInterrupted,
				Content:       content,
				CheckPointID:  checkpointID,
			}); err != nil {
				return err
			}
		}
		return nil
	}

	if action.BreakLoop != nil {
		return w.WriteEvent(ctx, &Event{
			Type:       EventTypeAction,
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: ActionTypeBreakLoop,
		})
	}

	if action.Exit {
		return w.WriteEvent(ctx, &Event{
			Type:       EventTypeAction,
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: ActionTypeExit,
		})
	}

	return nil
}

func formatRunPath(runPath []adk.RunStep) string {
	if len(runPath) == 0 {
		return ""
	}
	result := runPath[0].String()
	for _, step := range runPath[1:] {
		result += "/" + step.String()
	}
	return result
}
