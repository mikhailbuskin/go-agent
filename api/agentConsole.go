package main

// import (
// 	"bufio"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"os"
// 	"strings"

// 	"github.com/fatih/color"
// )

// func agentConsole() {

// 	// testing command
// 	// out, err := runCommand("ls -la")
// 	// if err != nil {
// 	// 	log.Fatal(err.Error())
// 	// }
// 	// fmt.Print(out)
// 	// return

// 	key := os.Getenv("OPENAI_API_KEY")

// 	ctx := context.TODO()

// 	// Tool definition
// 	tools := []ToolDef{
// 		{
// 			Type: "function",
// 			Function: FunctionDef{
// 				Name:        TOOL_RUN_COMMAND,
// 				Description: "Run a command in Ubuntu WSL shell",
// 				Parameters: map[string]interface{}{
// 					"type": "object",
// 					"properties": map[string]interface{}{
// 						"cmd": map[string]string{
// 							"type": "string",
// 						},
// 					},
// 					"required": []string{"cmd"},
// 				},
// 			},
// 		},
// 	}

// 	reader := bufio.NewReader(os.Stdin)

// 	/*messages example:
// 	[
// 	{role:user, message:"show me all files in the current folder"},
// 	// ai calls tool
// 	{role:assistant, message:"", "call_tools": [{},{}]},
// 	// error came
// 	{role:tool, message:"", "content":"tool1 result"},
// 	{role:tool, message:"", "content":"tool2 result"},
// 	// ai corrects tool call
// 	{role:assistant, message:"", "call_tools": [..]},
// 	// tool call returns result
// 	{role:tool, message:"", "content":"1.txt 2.txt"}
// 	// ai returns content
// 	{role:assistant, content:"Here are files which lays in the current folder."}
// 	]
// 	*/
// 	messages := []ChatMessage{}

// 	// Start of conversation loop
// 	for true {
// 		color.Set(color.FgHiBlue)

// 		fmt.Print("Enter prompt: ")
// 		prompt, _ := reader.ReadString('\n')
// 		prompt = strings.TrimSpace(prompt)
// 		//prompt := "Hello"

// 		color.Set(color.FgWhite)
// 		if prompt == "end" {
// 			break
// 		}

// 		// add user's prompt to message history
// 		messages = append(messages, ChatMessage{Role: "user", Content: prompt})
// 		//Say this is a test
// 		//List all files in the current directory using Ubuntu tool
// 		//Go back a folder from the current directory and tell me the contents of it

// 		i := 0

// 		// START OF TOOL LOOP
// 		for {

// 			// Step 1: Send user message
// 			req := ChatRequest{
// 				Model:    "gpt-4o-mini", // same as openai.ChatModelGPT4oMini
// 				Messages: messages,
// 				Tools:    tools,
// 				Stream:   true,
// 			}

// 			//chatResp, err := callOpenAI(ctx, key, req)
// 			chatResp, err := callOpenAIStream(ctx, key, req) // Change this to work
// 			if err != nil {
// 				log.Fatal(err)
// 			}

// 			agentmsg := chatResp.Choices[0].Message

// 			// add assistant message
// 			messages = append(messages, agentmsg)

// 			if len(agentmsg.ToolCalls) > 0 {
// 				for _, toolCall := range agentmsg.ToolCalls { //   _ --> index not needed
// 					if toolCall.Function.Name == TOOL_RUN_COMMAND {

// 						// arguments no longer double serialized so no need for this code
// 						// get the command
// 						var raw string
// 						err := json.Unmarshal(toolCall.Function.Arguments, &raw)
// 						if err != nil {
// 							log.Fatal(err)
// 						}

// 						var args struct {
// 							Cmd string `json:"cmd"`
// 						}

// 						err = json.Unmarshal([]byte(raw), &args)
// 						if err != nil {
// 							log.Fatal(err)
// 						}

// 						// run the command
// 						output, err := runCommand(args.Cmd)
// 						if err != nil {
// 							output = fmt.Sprintf("Error: %v", err)
// 						}

// 						tool_id := toolCall.ID

// 						messages = append(messages,
// 							ChatMessage{Role: "tool", Content: output, ToolCallID: tool_id},
// 						)
// 					}
// 				}
// 			} else {
// 				//fmt.Println(agentmsg.Content)
// 				break
// 			}

// 			i++
// 			if i > 10 {
// 				break
// 			}

// 		}
// 	}
// 	// END OF TOOL LOOP
// }
