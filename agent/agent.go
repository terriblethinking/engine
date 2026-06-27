package agent

import (
	"context"

	"fmt"
	"github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
	"slices"
)

type Agent struct {
	Client        bifrost.Bifrost
	Provider      schemas.ModelProvider
	Model         string
	Instructions  string
	Messages      []schemas.ChatMessage
	Tools         []schemas.ChatTool
	ToolWhitelist []string
	ToolBlacklist []string
}

// The text chunk the streaming agent outputs.
type AsyncAgentTextChunk struct {
	Content string
}

// The reasoning chunk the streaming agent outputs.
type AsyncAgentReasongingChunk struct {
	Index   int
	Type    string
	Content string
}

// The tool chunk the streaming agent outputs.
type AsyncAgentToolChunk struct {
	ID              string
	ToolName        string
	ToolParams      string
	ApproveToolChan chan bool
}

type AsyncAgentToolFailedChunk struct {
	ID         string
	ToolName   string
	ToolParams string
	Reason     string
}

// The error chunk a streaming agent can output.
type AsyncAgentErrorChunk struct {
	Content string
}

// All possible chunks an agent can output.
type AsyncAgentChunk interface {
	AsyncAgentTextChunk | AsyncAgentToolChunk | AsyncAgentReasongingChunk | AsyncAgentErrorChunk
}

func (a *Agent) Run(ctx context.Context, message string) (*schemas.BifrostChatResponse, *schemas.BifrostError) {
	response, err := a.Client.ChatCompletionRequest(schemas.NewBifrostContext(ctx, schemas.NoDeadline), &schemas.BifrostChatRequest{
		Provider: a.Provider,
		Model:    a.Model,
		Input: append(a.Messages,
			schemas.ChatMessage{
				Role: schemas.ChatMessageRoleUser,
				Content: &schemas.ChatMessageContent{
					ContentStr: schemas.Ptr(message),
				},
			}),
		Params: &schemas.ChatParameters{
			Tools: a.Tools,
		},
	})

	if err != nil {
		return response, err
	}

	return response, nil
}

func (a *Agent) RunAsync(ctx context.Context, message string) <-chan any {
	chunkChan := make(chan any, 100)
	// ApproveToolChan := make(chan bool, 5)

	// Run all agent stuff in a goroutine
	// to make it async.

	go func() {
		for {
			// Start a stream with Bifrost, using the agent settings and history.

			stream, err := a.Client.ChatCompletionStreamRequest(schemas.NewBifrostContext(context.Background(), schemas.NoDeadline), &schemas.BifrostChatRequest{
				Provider: a.Provider,
				Model:    a.Model,
				Input: append(a.Messages,
					schemas.ChatMessage{
						Role: schemas.ChatMessageRoleUser,
						Content: &schemas.ChatMessageContent{
							ContentStr: schemas.Ptr(message),
						},
					}),
			})

			if err != nil {
				chunkChan <- AsyncAgentErrorChunk{
					Content: err.String(),
				}
			}

			for chunk := range stream {
				if chunk.BifrostChatResponse != nil && len(chunk.BifrostChatResponse.Choices) > 0 {
					choice := chunk.BifrostChatResponse.Choices[0]

					// Check whether there is actual streaming content.

					if choice.ChatStreamResponseChoice != nil &&
						choice.ChatStreamResponseChoice.Delta != nil {

						// If the stream content is basic reasoning or response
						// text, return it to the client right away.

						if choice.ChatStreamResponseChoice.Delta.Content != nil {
							content := *choice.ChatStreamResponseChoice.Delta.Content
							chunkChan <- AsyncAgentTextChunk{
								Content: content,
							}
						}

						if choice.ChatStreamResponseChoice.Delta.Reasoning != nil {
							content := *choice.ChatStreamResponseChoice.Delta.Reasoning
							chunkChan <- AsyncAgentReasongingChunk{
								Content: content,
							}
						}

						// If the streaming content involves tools, then we have
						// a slightly different procedure.

						if len(choice.ChatStreamResponseChoice.Delta.ToolCalls) > 0 {
							for _, tool := range choice.ChatStreamResponseChoice.Delta.ToolCalls {
								if slices.Contains(a.ToolBlacklist, *tool.Function.Name) || slices.Contains(a.ToolBlacklist, "*") {

									// If the tool is in the blacklist (or client added "*"
									// to the blacklist, which automatically blacklists all
									// tools), we then check the whitelist. The whitelist
									// overrules the blacklist.

									if slices.Contains(a.ToolWhitelist, *tool.Function.Name) {

										// Create an ApproveToolChannel for each tool call,
										// which will be used to approve that specific tool.

										ApproveToolChannel := make(chan bool)

										toolChunk := AsyncAgentToolChunk{
											ID:              *tool.ID,
											ToolName:        *tool.Function.Name,
											ToolParams:      tool.Function.Arguments,
											ApproveToolChan: ApproveToolChannel,
										}

										chunkChan <- toolChunk

									} else {

										// If the tool is in the blacklist, and
										// not in the whitelist, then we send a
										// AsyncAgentToolFailedChunk error to the client
										// to tell them that the tool is blacklisted.

										toolBlacklistedErrorChunk := AsyncAgentToolFailedChunk{
											ID:         *tool.ID,
											ToolName:   *tool.Function.Name,
											ToolParams: *&tool.Function.Arguments,
											Reason:     "tool in blacklist.",
										}

										chunkChan <- toolBlacklistedErrorChunk

									}
								}
							}

						}
					}
				}
			}
		}
	}()

	return chunkChan
}
