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
	err := os.WriteFile(fileName, data, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
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

	folderName := "./chat"
	fileNameM := folderName + "/messages.json"
	fileNameH := folderName + "/history.json"

	messages := []ChatMessage{}
	history := []ChatMessage{}

	// Check if the folder exists
	_, err := os.Stat(fileNameM)

	if err != nil && !os.IsNotExist(err) { // if err isn't "doesn't exist"
		// Error checking file
		log.Fatal(err)
	}

	// If file doesn't exist -> create it
	if os.IsNotExist(err) {
		// File doesn't exist
		// Start new conversation

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

		// Create folder "chat"
		err = os.Mkdir(folderName, 0755) // 0755 sets permissions
		if err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			return
		}

		// Write "messages.json" file inside "chat" folder for messages to save
		updateMemory(&messages, fileNameM)
		updateMemory(&history, fileNameH)
	} else {
		// === CONTINUE CONVERSATION
		// Folder "chat" and File "messages.json" exists
		// Load existing messages memory

		// Read file
		contentsM, err := os.ReadFile(fileNameM) // contentsM is a JSON string in bytes
		if err != nil {
			log.Fatal(err)
		}
		// Load file into messages
		err = json.Unmarshal(contentsM, &messages)
		if err != nil {
			log.Fatal(err)
		}

		// Read file
		contentsH, err := os.ReadFile(fileNameH) // contentsH is a JSON string in bytes
		if err != nil {
			log.Fatal(err)
		}
		// Load file into messages
		err = json.Unmarshal(contentsH, &history)
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

		lastMes := messages[len(messages)-1]
		// last message is a user, assistant with tools, or a tool output message -> a continue is needed (user needs to press continue button)
		// Only acceptable thing to start a loop is for the last message to be an assistant without toolCalls
		if lastMes.Role == "user" || (lastMes.Role == "assistant" && len(lastMes.ToolCalls) > 0) || lastMes.Role == "tool" { // if there is an unfinished loop sequence (anything except assistant without toolCalls OR system message)
			sendWebMessage("error", "Continue needed because of an interrupted loop sequence", w)
			return
		}

		// add user's prompt to message history
		messages = append(messages, ChatMessage{Role: "user", Content: prompt})
		updateMemory(&messages, fileNameM)

		history = append(history, ChatMessage{Role: "user", Content: prompt})
		updateMemory(&history, fileNameH)
	} else {
		// CONTINUING UNFINISHED CYCLE

		lastMes := messages[len(messages)-1]
		// do not add any messages, continue the cycle (if something stopped).

		// if last message is assistant without tools, do nothing
		if lastMes.Role == "assistant" && len(lastMes.ToolCalls) == 0 {
			// exit loop because there is nothing to be continued on
			sendWebMessage("error", "Nothing to continue", w)
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

		// if last message is user, send to Open AI as is
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
			// Will then execute tools
		} else { // when last message is user or tool output to send back to the assistant (agent)
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
			updateMemory(&messages, fileNameM)

			// simplemsg := ChatMessage{Role: agentmsg.Role, Content: agentmsg.Content}
			// history = append(history, simplemsg)
			// updateMemory(&messages, fileNameH)
		}

		if len(agentmsg.ToolCalls) > 0 {
			for _, toolCall := range agentmsg.ToolCalls { //   _ --> index not needed
				if toolCall.Function.Name == TOOL_RUN_COMMAND {
					var raw string

					// arguments = "\"{\\\"cmd\\\":\\\"pwd\\\"}\"" (in bytes)
					// raw = "{\"cmd\": \"nikto -h www.maintstar.com -p 80\"}{\"cmd\": \"nikto -h www.maintstar.com -p 443\"}"
					err := json.Unmarshal(toolCall.Function.Arguments, &raw) // Undoing the double marshaled arguments
					if err != nil {
						log.Fatal(err)
					}

					//callmsg := fmt.Sprintf("Calling:\nType: %v : %v\nCommand : %v", toolCall.Type, toolCall.Function.Name, raw)
					call, err := json.Marshal(toolCall.Function) // equivalent to json.stringify(toolCall.Function) in App.jsx's memory loading
					if err != nil {
						log.Fatal(err)
					}
					callmsg := fmt.Sprintf("Calling:\n%v", string(call))
					sendWebMessage("assistant", callmsg, w)

					history = append(history, ChatMessage{Role: "assistant", Content: callmsg})
					updateMemory(&history, fileNameH)

					// breakArgs() Workflow :
					// "{...}{...}" => [{...},{...}]
					// extract command out of {...} execute it and append it to outputs[]. Repeat with the rest of the []
					// Convert outputs[] into one output : "Command 1 Output: ... \n Command 2 Output: ... \n ..."
					// return
					output, err := breakArgs(raw, w)
					if err != nil {
						log.Fatal(err)
					}

					//sendWebMessage("tool", output, w) // send message to web for display

					history = append(history, ChatMessage{Role: "tool", Content: output}) // add message to history variable
					updateMemory(&history, fileNameH)                                     // update history.json with new history

					tool_id := toolCall.ID

					messages = append(messages,
						ChatMessage{Role: "tool", Content: output, ToolCallID: tool_id},
					)
					updateMemory(&messages, fileNameM)

					// history = append(history,
					// 	ChatMessage{Role: "tool", Content: output},
					// )
					// updateMemory(&messages, fileNameH)
				}
			}
		} else {
			//fmt.Println(agentmsg.Content)
			// err := sendWebMessage("---end---", w)
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// add final assistant message (without tools) to history
			simplemsg := ChatMessage{Role: "assistant", Content: agentmsg.Content}
			history = append(history, simplemsg) // add message to history variable
			updateMemory(&history, fileNameH)    // update history.json with new history

			break
		}

		lastAssisWTools = false
		//lastIsTool = false

		i++
		if i > 10 {
			err := sendWebMessage("error", "Loop Run Limit Exceeded", w)
			if err != nil {
				log.Fatal(err)
			}
			break
		}

	}
	//}
	// END OF TOOL LOOP
}

func breakArgs(raw string, w http.ResponseWriter) (string, error) {
	// raw = "{\"cmd\": \"nikto -h www.maintstar.com -p 80\"}{\"cmd\": \"nikto -h www.maintstar.com -p 443\"}"
	output := ""

	var raws []string
	begin := 0
	for i := 0; i < (len(raw) - 1); i++ { // Go through indexes 0 to the 2nd to last one
		if raw[i] == '}' && raw[i+1] == '{' {
			raws = append(raws, raw[begin:(i+1)])
			begin = i + 1
		}
	}
	raws = append(raws, raw[begin:])

	// raws = [ {\"cmd\": \"nikto -h www.maintstar.com -p 80\"}, {\"cmd\": \"nikto -h www.maintstar.com -p 443\"} ]

	var args struct {
		Cmd string `json:"cmd"`
	}

	//rawsLength := len(raws)
	//var outputs []string

	for i := range raws {
		err := json.Unmarshal([]byte(raws[i]), &args)
		if err != nil {
			return "", err
		}

		msg := fmt.Sprintf("Command %d Output: \n", i+1)
		output += msg
		sendWebMessage("tool", msg, w)

		// run the command
		tempOut, exitCode, err := runCommand(args.Cmd, w) // Calls sendWebMessage() within it to stream command output by line

		output += tempOut

		if err != nil {
			// outputs[i] = fmt.Sprintf("Error: %v", err)
			// break
			return "", err // ---------------------------stops all the commands because if this one doesn't work then the next ones probably won't. Need to add which command failed
		}

		exitMsg := fmt.Sprintf("\nCommand %d Exit Code: %d", i+1, exitCode)

		if i < len(raws)-1 { // if it is not the iteration then make space
			exitMsg += "\n\n"
		}

		sendWebMessage("tool", exitMsg, w)
		output += exitMsg
	}

	// outputs = [ "(Output 1) Exit Code: (Exit Code)", "(Output 2) Exit Code: (Exit Code)" ]

	// var output string
	// for i := range outputs {
	// 	output += fmt.Sprintf("Command %d Output: \n%s", (i + 1), outputs[i])
	// 	if i < len(outputs)-1 {
	// 		output += "\n\n"
	// 	} // if it is not the last Output then make space
	// }

	// output = "Command 1 Output: ... \n Exit Code: (Exit Code) \n Command 2 Output: ... \n Exit Code: (Exit Code)"

	return output, nil
}
