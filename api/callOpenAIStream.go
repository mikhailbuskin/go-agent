package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func runCommand(cmd string) (string, error) {
	// Run in docker container
	command := "docker exec kali " + cmd // add docker exec kali to run the cmd in an environment container
	out, err := exec.Command("bash", "-c", command).CombinedOutput()
	return string(out), err
}

type ChatMessage struct {
	Role    string `json:"role"`    // role: "user" / "assistant" / "tool"
	Content string `json:"content"` // can't be omitempty, required field
	//ToolCall   *ToolCall  `json:"tool_call,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // this field is only needed and used when role:"assistant"
	ToolCallID string     `json:"tool_call_id,omitempty"` // this field is only needed and used when role:"tool"
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
	Stream   bool          `json:"stream"`
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
	//Message ChatMessage `json:"message"`
	Message ChatMessage `json:"delta"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

const TOOL_RUN_COMMAND = "run_command"

func callOpenAIStream(ctx context.Context, apiKey string, reqBody ChatRequest, w http.ResponseWriter) (*ChatResponse, error) { //With streaming there is no ChatResponce being returned at once
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

	var chatResp ChatResponse

	//storing the streaming data
	var content string
	var argument string
	// f, err := os.OpenFile("response.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// defer f.Close()

	areToolCalls := false

	errMes := ""
	errCount := 0

	i := 0

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		//os.WriteFile("response.txt", []byte(line+"\n"), os.ModeAppend)

		// Skip empty lines & [DONE]
		if line == "" {
			continue
		} else if line == "data: [DONE]" {
			break
		}
		// if len(line) > 6 && line[:6] == "data: " {
		// 	line = line[6:]
		// }

		// Streaming API Response
		// data: {"id":"...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"}}]}
		// data: {"id":"...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"}}]}
		// data: {"id":"...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"}}]}
		// ...
		// data: [DONE]

		if !(len(line) > 6 && line[:6] == "data: ") {
			errMes += line
			errCount++
			if errCount > 10 {
				log.Fatal(errMes)
			}
			continue
		}
		line = line[6:]

		//var formated_line map[string]any
		// err := json.Unmarshal([]byte(line), &formated_line)
		// data, _ := json.MarshalIndent(formated_line, "", " ")
		// os.WriteFile("output.json", data, 0644)

		// easier to read format of assistent output
		var data StreamResponse
		err := json.Unmarshal([]byte(line), &data)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(data)

		// var chunk struct {
		// 	Choices []struct {
		// 		Delta struct {
		// 			Content   string `json:"content"`
		// 			ToolCalls string `json:tool_calls`
		// 		} `json:"delta"`
		// 	} `json:"choices"`
		// }

		//forming the Response to later use for message history
		if areToolCalls { // this also works as i!=0 because areToolCalls is always equal to false the first time over
			chatResp.Choices[0].Message.ToolCalls[0].Function.Arguments = nil
		}
		// chatResp.Choices[0].Message.ToolCalls = []ToolCall{
		// 	{
		// 		Function: ToolFunction{
		// 			Arguments: []byte("{}"),
		// 		},
		// 	},
		// }

		if len(chatResp.Choices) > 0 {
			chatResp.Choices[0].Message.Content = ""
		}

		err = json.Unmarshal([]byte(line), &chatResp)
		if err != nil {
			return nil, err
		}

		if len(chatResp.Choices[0].Message.ToolCalls) > 0 {
			areToolCalls = true
		}

		if len(chatResp.Choices) > 0 {
			delta := chatResp.Choices[0].Message
			content += delta.Content
			if areToolCalls {
				argsStr := ""
				_ = json.Unmarshal(delta.ToolCalls[0].Function.Arguments, &argsStr)
				argument += argsStr
			}
			if delta.Content != "" {
				fmt.Print(delta.Content) // prints piece by piece
				// Send the message
				err := sendWebMessage(delta.Content, w)
				if err != nil {
					return nil, err
				}
			}
		}

		i++
	}
	// chatResp.Choices[0].Message.Role = "assistant"
	chatResp.Choices[0].Message.Content = content

	farg, err := json.Marshal(argument)
	if err != nil {
		return nil, err
	}

	if areToolCalls {
		chatResp.Choices[0].Message.ToolCalls[0].Function.Arguments = farg
	}

	// chatResp := ChatResponse{
	// 	Choices: []Choice{
	// 		{ // [0]
	// 			Message: ChatMessage{
	// 				Role:    "assistant",
	// 				Content: content,
	// 			},
	// 		},
	// 	},
	// }

	if content != "" {
		// color.Set(color.FgYellow)
		// fmt.Print("\n--- stream done ---")
		// color.Set(color.FgWhite)
		fmt.Println()
	}
	return &chatResp, nil
}
