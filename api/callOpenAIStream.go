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

func oldrunCommand(cmd string) (string, int, error) {
	// Run in docker container
	// Use this format to run command:
	// docker exec kali bash -c "[COMMAND]"
	command := "docker exec kali bash -c \"" + cmd + "\"" // add docker exec kali to run the cmd in an environment container
	out, err := exec.Command("bash", "-c", command).CombinedOutput()

	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			exitCode = exitErr.ExitCode()
			// return output and exit code (no Go 'error' unless it's something else)
			return string(out), exitCode, nil
		}
		// failed to start or something else
		return string(out), -1, err
	}

	return string(out), exitCode, err
}

func runCommand(command string, w http.ResponseWriter) (string, int, error) { // Returns final output, exit error, and error message (nil if no errors)
	// Run shell command in docker container
	// Use this format to run cmdStr:
	// docker exec kali bash -c "[COMMAND]"
	var fOut bytes.Buffer

	cmdStr := "docker exec kali bash -c \"" + command + "\"" // add docker exec kali to run the cmd in an environment container
	cmd := exec.Command("bash", "-c", cmdStr)

	// // merge stderr into stdout
	// cmd.Stderr = cmd.Stdout

	// // single unified pipe
	// pipe, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return "", -1, err
	// }

	// err = cmd.Start()
	// if err != nil {
	// 	return "", -1, err
	// }

	// // stream unified output
	// scanner := bufio.NewScanner(pipe)
	// for scanner.Scan() {
	// 	out := scanner.Text()
	// 	fOut += out
	// 	fmt.Print(out) // -------------------------------Replace with sendMessage() to web
	// 	//sendWebMessage("tool", out, w)
	// }

	// err = cmd.Wait()

	// Connect pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", -1, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", -1, err
	}

	// Start command before reading pipes
	if err := cmd.Start(); err != nil {
		return "", -1, err
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fOut.WriteString(line + "\n")
			fmt.Println(line)
			sendWebMessage("tool", line+"\n", w)
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fOut.WriteString(line + "\n")
			fmt.Println(line)
			sendWebMessage("tool", line, w)
		}
	}()

	// Wait until command finishes
	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError) // ok is true when the err is an exit error (non fatal) and false when something other than a fatal error
		if ok {                              // If the err is just an exit error
			exitCode = exitErr.ExitCode() // retrieving the exit code of the exit error
			// return output and exit code (no Go 'error' unless it's something else except an exit error)
			return fOut.String(), exitCode, nil // Don't return it as a fatal error
		}
		// failed to start or something else. Not an exit error but another error
		return fOut.String(), -1, err // Return the err to Fatal()
	}

	return fOut.String(), exitCode, err // error = nil. exitCode = 0.
}

// each message in messages.json and history.json
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
			if errCount > 100 { // Just for safety. It is usually going to break itself through the if statements above once the error message ends
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
				err := sendWebMessage("assistant", delta.Content, w)
				if err != nil {
					return nil, err
				}
			}
		}

		i++
	}
	if errCount > 0 {
		log.Fatal(errMes)
	}
	// chatResp.Choices[0].Message.Role = "assistant"
	chatResp.Choices[0].Message.Content = content

	// argument = "{\"cmd\":\"pwd\"}"
	farg, err := json.Marshal(argument) // Stringifying it a second time for it to be readable by the agent
	// farg = "\"{\\\"cmd\\\":\\\"pwd\\\"}\"" (in bytes)
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
