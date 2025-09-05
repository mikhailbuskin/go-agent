package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

func runCommand(cmd string) (string, error) {
	// Run in Ubuntu/WSL shell
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
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
	err = json.Unmarshal(data, &chatResp)
	if err != nil {
		return nil, err
	}
	return &chatResp, nil
}

func callOpenAIStream(ctx context.Context, apiKey string, reqBody ChatRequest) (*ChatResponse, error) { //With streaming there is no ChatResponce being returned at once
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
		if areToolCalls == true {
			chatResp.Choices[0].Message.ToolCalls[0].Function.Arguments = nil
		}
		// chatResp.Choices[0].Message.ToolCalls = []ToolCall{
		// 	{
		// 		Function: ToolFunction{
		// 			Arguments: []byte("{}"),
		// 		},
		// 	},
		// }

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
			if areToolCalls == true {
				argsStr := ""
				_ = json.Unmarshal(delta.ToolCalls[0].Function.Arguments, &argsStr)
				argument += argsStr
			}
			fmt.Print(delta.Content) // prints piece by piece
		}

		i++
	}
	// chatResp.Choices[0].Message.Role = "assistant"
	chatResp.Choices[0].Message.Content = content

	farg, err := json.Marshal(argument)
	if err != nil {
		return nil, err
	}

	if areToolCalls == true {
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
		fmt.Println("\n--- stream done ---")
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

	reader := bufio.NewReader(os.Stdin)

	// Start of conversation loop
	for true {
		//prompt := ""

		fmt.Print("Enter prompt: ")
		//fmt.Scanln(&prompt)
		//prompt = "Say this is a test"
		prompt, _ := reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)

		if prompt == "end" {
			break
		}

		// start message history
		messages := []ChatMessage{
			{Role: "user", Content: prompt},
		}
		//Say this is a test
		//List all files in the current directory using Ubuntu tool
		//Go back a folder from the current directory and tell me the contents of it

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

		i := 0

		// START OF TOOL LOOP
		for true {

			// Step 1: Send user message
			req := ChatRequest{
				Model:    "gpt-4o-mini", // same as openai.ChatModelGPT4oMini
				Messages: messages,
				Tools:    tools,
				Stream:   true,
			}

			//chatResp, err := callOpenAI(ctx, key, req)
			chatResp, err := callOpenAIStream(ctx, key, req)
			if err != nil {
				log.Fatal(err)
			}

			agentmsg := chatResp.Choices[0].Message

			// add assistant message
			messages = append(messages, agentmsg)

			//break

			if len(agentmsg.ToolCalls) > 0 {
				for _, toolCall := range agentmsg.ToolCalls { //   _ --> index not needed
					if toolCall.Function.Name == TOOL_RUN_COMMAND {

						// arguments no longer double serialized so no need for this code
						// get the command
						var raw string
						err := json.Unmarshal(toolCall.Function.Arguments, &raw)
						if err != nil {
							log.Fatal(err)
						}

						var args struct {
							Cmd string `json:"cmd"`
						}

						//err = json.Unmarshal(toolCall.Function.Arguments, &args)
						err = json.Unmarshal([]byte(raw), &args)
						//err = json.Unmarshal(err)
						if err != nil {
							log.Fatal(err)
						}

						// run the command
						output, err := runCommand(args.Cmd)
						if err != nil {
							output = fmt.Sprintf("Error: %v", err)
						}

						tool_id := toolCall.ID

						messages = append(messages,
							ChatMessage{Role: "tool", Content: output, ToolCallID: tool_id},
						)
					}
				}
			} else {
				//fmt.Println(agentmsg.Content)
				break
			}

			i++
			if i > 10 {
				break
			}

		}
	}
	// END OF TOOL LOOP
}
