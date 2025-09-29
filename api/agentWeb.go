package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func updateMemory(messages *[]ChatMessage, fileName string) {
	data, _ := json.MarshalIndent(messages, "", " ") // json.MarshalIndent (and json.Marshal) can take either a value or a pointer to a value/structure. It doesn’t require the value specifically.
	os.WriteFile(fileName, data, 0644)
}

func agentWeb(prompt string, w http.ResponseWriter) {

	// testing command
	// out, err := runCommand("ls -la")
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }
	// fmt.Print(out)
	// return

	key := os.Getenv("OPENAI_API_KEY")

	ctx := context.TODO()

	// Tool definition
	tools := []ToolDef{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        TOOL_RUN_COMMAND,
				Description: "Run a command in Kali Linux environment",
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

	/*messages example:
	[
	{role:user, message:"show me all files in the current folder"},
	// ai calls tool
	{role:assistant, message:"", "call_tools": [{},{}]},
	// error came
	{role:tool, message:"", "content":"tool1 result"},
	{role:tool, message:"", "content":"tool2 result"},
	// ai corrects tool call
	{role:assistant, message:"", "call_tools": [..]},
	// tool call returns result
	{role:tool, message:"", "content":"1.txt 2.txt"}
	// ai returns content
	{role:assistant, content:"Here are files which lays in the current folder."}
	]
	*/

	fileName := "messages.json"

	messages := []ChatMessage{}

	// Check if file exists.
	_, err := os.Stat(fileName)
	// === NEW CONVERSATION
	if err != nil {
		// File doesn't exit
		// Load starter template

		// var formated_line map[string]any
		// err := json.Unmarshal([]byte(line), &formated_line)

		//read starter messages
		startmessagebyte, err := os.ReadFile("start_messages.txt")
		startmessagestr := string(startmessagebyte)
		if err != nil {
			log.Fatal(err)
		}

		// Load file into messages
		messages = []ChatMessage{{Role: "system", Content: startmessagestr}}

		// Write file for messages to save
		updateMemory(&messages, fileName)
	} else {
		// === CONTINUE CONVERSATION
		// File exists
		// Load existing messages memory

		// Read file
		contents, err := os.ReadFile(fileName)
		if err != nil {
			log.Fatal(err)
		}
		// Load file into messages
		err = json.Unmarshal(contents, &messages)
		if err != nil {
			log.Fatal(err)
		}
	}

	// randkey := (uuid.New()).String() // generates a random UUID (version 4)

	// randkey = randkey

	// Start of conversation loop
	//for true {

	// if prompt == "end" {
	// 	break
	// }

	lastAssisWTools := false
	//lastIsTool := false

	if prompt != "" {
		// STARTING CYCLE

		// add user's prompt to message history
		messages = append(messages, ChatMessage{Role: "user", Content: prompt})
		updateMemory(&messages, fileName)
	} else {
		// CONTINUING UNFINISHED CYCLE

		lastMes := messages[len(messages)-1]
		// do not add any messages, continue the cycle (if something stopped).

		// if last message is assistant without tools, do nothing
		if lastMes.Role == "assistant" && len(lastMes.ToolCalls) == 0 {
			// exit loop because there is nothing to be continued on
			return // --------------------------------------------------------------------------------------------------------------------------------------------------------------------pay attention to this, need to send back "Nothing to continue"
		}

		// if last message is assistant with tools, call tools.
		if lastMes.Role == "assistant" && len(lastMes.ToolCalls) > 0 {
			// call tool and go from there
			lastAssisWTools = true
		}
		// if last message is tool, send to OpenAI as is
		//if lastMes.Role == "tool" {
		// send to messages to OpenAI to review the command output and go from there
		// so just do the regular as you would do with lastMessage being user, don't need a variable

		//lastIsTool = true
		//}
	}

	//Sample Prompts:
	//Say this is a test
	//List all files in the current directory using Ubuntu tool
	//Go back a folder from the current directory and tell me the contents of it

	i := 0

	// START OF TOOL LOOP
	for {
		var agentmsg ChatMessage

		if lastAssisWTools { // When last message is assistant with tools
			agentmsg = messages[len(messages)-1] // set as the last message
		} else { // when last message is tool output to send back to the assistant or user
			// Step 1: Send user message
			req := ChatRequest{
				Model:    "gpt-4o-mini", // same as openai.ChatModelGPT4oMini
				Messages: messages,
				Tools:    tools,
				Stream:   true,
			}

			//chatResp, err := callOpenAI(ctx, key, req)
			chatResp, err := callOpenAIStream(ctx, key, req, w)
			if err != nil {
				log.Fatal(err)
			}

			agentmsg = chatResp.Choices[0].Message

			// add assistant message
			messages = append(messages, agentmsg)
			updateMemory(&messages, fileName)
		}

		if len(agentmsg.ToolCalls) > 0 {
			for _, toolCall := range agentmsg.ToolCalls { //   _ --> index not needed
				if toolCall.Function.Name == TOOL_RUN_COMMAND {

					var raw string

					// raw = "{\"cmd\": \"nikto -h www.maintstar.com -p 80\"}{\"cmd\": \"nikto -h www.maintstar.com -p 443\"}"
					err := json.Unmarshal(toolCall.Function.Arguments, &raw)
					if err != nil {
						log.Fatal(err)
					}

					// breakArgs() Workflow :
					// {...}{...} => [{...}{...}]
					// extract command out of {...} execute it and append it to outputs[]. Repeat with the rest of the []
					// Convert outputs[] into one output : "Command 1 Output: ... \n Command 2 Output: ... \n ..."
					// return
					output, err := breakArgs(raw)
					if err != nil {
						log.Fatal(err)
					}

					tool_id := toolCall.ID

					messages = append(messages,
						ChatMessage{Role: "tool", Content: output, ToolCallID: tool_id},
					)
					updateMemory(&messages, fileName)
				}
			}
		} else {
			//fmt.Println(agentmsg.Content)
			// err := sendWebMessage("---end---", w)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			break
		}

		lastAssisWTools = false
		//lastIsTool = false

		i++
		if i > 10 {
			err := sendWebMessage("Loop Run Limit Exceeded", w)
			if err != nil {
				log.Fatal(err)
			}
			break
		}

	}
	//}
	// END OF TOOL LOOP
}

func breakArgs(raw string) (string, error) {
	var raws []string
	begin := 0
	for i := 0; i < (len(raw) - 1); i++ { // Go through indexes 0 to the 2nd to last one
		if raw[i] == '}' && raw[i+1] == '{' {
			raws = append(raws, raw[begin:(i+1)])
			begin = i + 1
		}
	}
	raws = append(raws, raw[begin:])

	var args struct {
		Cmd string `json:"cmd"`
	}

	//rawsLength := len(raws)
	var outputs []string

	for i := range raws {
		err := json.Unmarshal([]byte(raws[i]), &args)
		if err != nil {
			return "", err
		}

		// run the command
		tempOut, exitCode, err := runCommand(args.Cmd)

		tempOut += fmt.Sprintf("\nExit Code: %d", exitCode)

		outputs = append(outputs, tempOut)

		if err != nil {
			// outputs[i] = fmt.Sprintf("Error: %v", err)
			// break
			return "", err
		}
	}

	var output string
	for i := range outputs {
		output += fmt.Sprintf("Command %d Output: %s\n\n", (i + 1), outputs[i])
	}

	return output, nil
}
