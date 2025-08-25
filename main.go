package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

func runCommand(cmd string) (string, error) {
	// Run in Ubuntu/WSL shell
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	return string(out), err
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"` // can't be omitempty, required field
	//ToolCall   *ToolCall  `json:"tool_call,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []ToolDef     `json:"tools,omitempty"`
}

type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Choice struct {
	Message ChatMessage `json:"message"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

const TOOL_RUN_COMMAND = "run_command"

func callOpenAI(ctx context.Context, apiKey string, reqBody ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", data)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(data, &chatResp); err != nil {
		return nil, err
	}
	return &chatResp, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// test command
	// out, err := runCommand("ls -la")
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }
	// fmt.Print(out)
	// return

	key := os.Getenv("OPENAI_API_KEY")

	ctx := context.TODO()

	messages := []ChatMessage{
		{Role: "user", Content: "List all files in the current directory using Ubuntu tool"},
	}

	// Tool definition
	tools := []ToolDef{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        TOOL_RUN_COMMAND,
				Description: "Run a command in Ubuntu WSL shell",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"cmd": map[string]string{
							"type": "string",
						},
					},
					"required": []string{"cmd"},
				},
			},
		},
	}

	// Step 1: Send user message
	req := ChatRequest{
		Model:    "gpt-4o", // same as openai.ChatModelGPT4o
		Messages: messages,
		Tools:    tools,
	}

	chatResp, err := callOpenAI(ctx, key, req)
	if err != nil {
		log.Fatal(err)
	}

	agentmsg := chatResp.Choices[0].Message

	// Step 2: Check for tool calls
	if len(agentmsg.ToolCalls) > 0 {
		for _, toolCall := range agentmsg.ToolCalls { //   _ --> index not needed
			if toolCall.Function.Name == TOOL_RUN_COMMAND {

				var raw string
				err := json.Unmarshal(toolCall.Function.Arguments, &raw)
				// if err == nil {
				// 	// It was double-encoded, so now unmarshal raw
				// 	toolCall.Function.Arguments = json.RawMessage(raw)
				// }
				if err != nil {
					panic(err) // switch to Fatal
				}

				var args struct {
					Cmd string `json:"cmd"`
				}

				err = json.Unmarshal([]byte(raw), &args) // toolCall.Function.Arguments = "\"{\\\"cmd\\\":\\\"ls\\\"}\""
				//err = json.Unmarshal(err)
				if err != nil {
					panic(err) // switch to Fatal
				}

				output, err := runCommand(args.Cmd)
				if err != nil {
					output = fmt.Sprintf("Error: %v", err)
				}

				tool_id := toolCall.ID

				// Send tool output back
				messages = append(messages,
					ChatMessage{Role: "assistant", Content: agentmsg.Content, ToolCalls: []ToolCall{toolCall}},
					ChatMessage{Role: "tool", Content: output, ToolCallID: tool_id},
				)

				chatResp, err = callOpenAI(ctx, key, ChatRequest{
					Model:    "gpt-4o",
					Messages: messages,
				})
				// json_str, err := json.MarshalIndent(messages, "", "  ")
				// os.WriteFile("output.json", []byte(json_str), 0644)
				if err != nil {
					panic(err) // *errors.errorString {s: "API error: {\n  \"error\": {\n    \"message\": \"Invalid parameter: messages with role 'tool' must be a response to a preceeding message with 'tool_calls'.\",\n    \"type\": \"invalid_request_error\",\n    \"param\": \"messages.[2].role\",\n    \"code\": null\n  }\n}"}
				}

				fmt.Println(chatResp.Choices[0].Message.Content)
			}
		}
	} else {
		fmt.Println(agentmsg.Content)
	}
}
