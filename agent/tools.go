package agent

import (
	"strings"

	bifrost "github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
)

type ToolFuncResponse struct {
	Message string
	Result  any
}

type ToolChunk struct {
	ID              string
	ToolName        string
	ToolParams      string
	ApproveToolChan chan bool
}

type ToolFailedChunk struct {
	ID         string
	ToolName   string
	ToolParams string
	Reason     string
}

type ToolFunc func(...any) (ToolFuncResponse, error)

type Tools map[string]ToolFunc

// ToolManager is the function that... well, manages tool
// calls. It recieves the tool chunk (for the tool information),
// the tool function (to be able to run the tool), and the
// chunk channel to be able to communicate with the main
// go routine.

func ToolManager(ctx schemas.BifrostContext, tool ToolChunk, toolFunc ToolFunc, toolCall schemas.ChatAssistantMessageToolCall, chunkChan chan any) {

	// Here we use the ApproveToolChan located inside the ToolChunk
	// to wait for the the user to confirm (or reject) the tool.

	select {
	case approved := <-tool.ApproveToolChan:
		if approved {

			// If the user approves the tool, then we run the tool
			// function.

			// Set the response and err.

			var response any
			var err error

			// Here we check whether a tool is an MCP tool or not.
			// This will tell us whether to execute it as an actual
			// function or through bifrost.ExecuteChatMCPTool.
			// The way we check is whether the tool name contains a "-".
			// This is kinda bad and unweildy, as we have to ban users
			// from creating any tool names with the character "-",
			// but there is (currently) no other way that I have found.

			if strings.Contains(tool.ToolName, "-") {
				response, err = bifrost.ExecuteChatMCPTool(ctx, toolCall)
			} else {
				response, err = toolFunc(tool.ToolParams)
			}

			// If there is an error, send it to the client through
			// the chunk chan.

			if err != nil {
				chunkChan <- ToolFailedChunk{
					ID:         tool.ID,
					ToolName:   tool.ToolName,
					ToolParams: tool.ToolParams,
					Reason:     err.Error(),
				}

				return
			}

			// If there is no error, then we send the tool response
			// back to the client, where is can be further managed
			// (such as add to chat history for the model to see).

			chunkChan <- response

			return
		} else {

			// If the user rejected the tool, then we simply send
			// a ToolFailedChunk with Reason of "tool rejected".

			chunkChan <- ToolFailedChunk{
				ID:         tool.ID,
				ToolName:   tool.ToolName,
				ToolParams: tool.ToolParams,
				Reason:     "tool rejected",
			}

			return
		}
	}
}
