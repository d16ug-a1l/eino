package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/web"
)

// echoAgent is a simple demo agent that echoes user input.
type echoAgent struct{}

func (a *echoAgent) Name(_ context.Context) string          { return "EinoDemo" }
func (a *echoAgent) Description(_ context.Context) string   { return "A demonstration agent" }
func (a *echoAgent) Run(_ context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	var query string
	if len(input.Messages) > 0 {
		query = input.Messages[len(input.Messages)-1].Content
	}

	go func() {
		if input.EnableStreaming {
			stream := schema.StreamReaderFromArray([]*schema.Message{
				{Role: schema.Assistant, Content: "你好！我是 Eino Demo Agent。\n"},
				{Role: schema.Assistant, Content: "你刚才说的是："},
				{Role: schema.Assistant, Content: query},
			})
			gen.Send(adk.EventFromMessage(nil, stream, schema.Assistant, ""))
		} else {
			gen.Send(adk.EventFromMessage(
				schema.AssistantMessage("echo: "+query, nil),
				nil, schema.Assistant, ""))
		}
		gen.Send(&adk.AgentEvent{Action: adk.NewExitAction()})
		gen.Close()
	}()
	return iter
}

func main() {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &echoAgent{},
		EnableStreaming: true,
	})

	srv := web.NewServer(runner, web.WithAddr(":8080"))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Starting Eino web server on http://localhost:8080")
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
