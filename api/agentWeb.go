package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

func updateMemory(messages *[]ChatMessage, fileName string) {
	data, _ := json.MarshalIndent(messages, "", " ") // json.MarshalIndent (and json.Marshal) can take either a value or a pointer to a value/structure. It doesn’t require the value specifically.
	err := os.WriteFile(fileName, data, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
}

func agentWeb(prompt string, w http.ResponseWriter) {

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
							"type":        "string",
							"description": "Shell command to execute in the Kali Linux environment",
						},
						"intent": map[string]interface{}{
							"type":        "string",
							"description": "Why this command is being run",
						},
					},
					"required": []string{"cmd", "intent"},
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

	folderName := "./chats/chat"
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

		//read system message
		systemmessagebyte, err := os.ReadFile("system_message.txt")
		systemmessage := string(systemmessagebyte)
		if err != nil {
			log.Fatal(err)
		}

		// Load file into messages
		messages = []ChatMessage{{Role: "system", Content: systemmessage}}

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

	lastAssisWTools := false

	// Checking if user pressed "continue" or "send" button
	if prompt != "" {
		// STARTING CYCLE

		lastMes := messages[len(messages)-1]
		// last message is a user, assistant with tools, or a tool output message -> a continue is needed (user needs to press continue button)
		// Only acceptable thing to start a loop is for the last message to be an assistant without toolCalls
		if lastMes.Role == "user" || (lastMes.Role == "assistant" && len(lastMes.ToolCalls) > 0) || lastMes.Role == "tool" { // if there is an unfinished loop sequence (anything except assistant without toolCalls OR system message)
			sendWebMessage("error", "Continue needed because of an interrupted loop sequence", w)
			return
		}

		// trim history.json in case program has stopped in between the update memories---------------if a lot of customer, make sure this isn't a problem
		// add user's prompt to message history
		history = append(history, ChatMessage{Role: "user", Content: prompt})
		messages = append(messages, ChatMessage{Role: "user", Content: prompt})

		updateMemory(&history, fileNameH)
		updateMemory(&messages, fileNameM)

	} else {
		// CONTINUING UNFINISHED CYCLE

		lastMes := messages[len(messages)-1]
		// do not add any messages, continue the cycle (if something stopped).

		// if last message is assistant without tools, do nothing
		if (lastMes.Role == "assistant" && len(lastMes.ToolCalls) == 0) || (lastMes.Role == "system") {
			// exit loop because there is nothing to be continued on
			sendWebMessage("error", "Nothing to continue", w)
			return
		}

		// if last message is assistant with tools, call tools.
		if lastMes.Role == "assistant" && len(lastMes.ToolCalls) > 0 {
			// call tool and go from there
			lastAssisWTools = true
		}
		// if last message is tool, send to OpenAI as is (it does the same thing in the regular cycle. If received toolCall -> call command -> return back to agent)

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

			chatResp, err := callOpenAIStream(ctx, key, req, w, "assistant") // streams content to website inside of it and returns the final result.
			if err != nil {
				log.Fatal(err)
			}

			agentmsg = chatResp.Choices[0].Message

			// add assistant message
			messages = append(messages, agentmsg)

			// when assistant with a response.
			if agentmsg.Content != "" { // can be an assistant with toolCall AND a response
				if len(agentmsg.ToolCalls) > 0 {
					sendWebMessage("break", "", w) // makes sure that the two assistant messages are not together (the content and the toolCall). Does not effect history but only streaming.
				}

				simplemsg := ChatMessage{Role: "assistant", Content: agentmsg.Content}
				history = append(history, simplemsg)
			}

			// update together so they don't go out of sync
			updateMemory(&history, fileNameH) // will only do something if there is content
			updateMemory(&messages, fileNameM)
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

					// breakArgs() Workflow :
					// "{...}{...}" => [{...},{...}]
					// extract command out of {...} execute it and append it to outputs[]. Repeat with the rest of the []
					// Convert outputs[] into one output : "Command 1 Output: ... \n Command 2 Output: ... \n ..."
					// return
					output, outputs, err := breakArgs(raw, w, &history) // streams the command outputs inside of it progressively.
					outputs = outputs                                   // outputs not used at this moment so to not get error that it is not being used I do this
					if err != nil {
						log.Fatal(err)
					}
					history = append(history, ChatMessage{Role: "tool", Content: output}) // add unsummarized cmd output to history
					sendWebMessage("break", "", w)                                        // Breaks apart the raw command (tool) output and the shortened command (tool) output

					// sumarizeOut() Workflow:
					// Create a messages array with these contents:
					//     Custom system message
					//     Assistant message (with toolCall command and intent) (but without any content)
					//     Tool message (command output)
					//     User message "Summarize above cmd output keeping everything that may be useful based on the intent"
					shortenedOut := output // set it to the raw output

					tokensNum := countTokens(output)
					if tokensNum > 700 {
						shortenedOut = summarizeOut(output, toolCall, w)                            // change it to the summarized output if it is too long
						history = append(history, ChatMessage{Role: "tool", Content: shortenedOut}) // add summarized cmd output to history
					} else {
						nosummsg := "No summarization needed. Under 700 tokens"
						sendWebMessage("tool", nosummsg, w)
						history = append(history, ChatMessage{Role: "tool", Content: nosummsg}) // add unsummarized cmd output to history
					}
					//shortenedOut = "   ***Summarized ToolCall Output***\n\n" + shortenedOut

					//sendWebMessage("tool", shortenedOut, w)

					tool_id := toolCall.ID
					messages = append(messages,
						ChatMessage{Role: "tool", Content: shortenedOut, ToolCallID: tool_id}, // add summarized cmd output to messages
					)

					updateMemory(&history, fileNameH) // update history.json with new history
					updateMemory(&messages, fileNameM)
				}
			}
		} else { // when the response is without any toolCalls. Just an "assistant" message -> break the tool loop
			break
		}

		lastAssisWTools = false

		i++
		if i > 30 {
			err := sendWebMessage("error", "Loop Run Limit Exceeded", w)
			if err != nil {
				log.Fatal(err)
			}
			break
		}

	}
	// END OF TOOL LOOP
}

// Returns:
// output ("Command 1 Output: ... \n Exit Code: (Exit Code) \n Command 2 Output: ... \n Exit Code: (Exit Code)")
// outputs ([...],[...]) (Without the commentary like "Command 1 Output:"), and
// Error
func breakArgs(raw string, w http.ResponseWriter, history *[]ChatMessage) (string, []string, error) {
	// raw = "{\"cmd\": \"nikto -h www.maintstar.com -p 80\"}{\"cmd\": \"nikto -h www.maintstar.com -p 443\"}"
	output := ""
	outputs := []string{}

	var raws []string
	begin := 0
	for i := 0; i < (len(raw) - 1); i++ { // Go through indexes 0 to the 2nd to last one
		if raw[i] == '}' && raw[i+1] == '{' {
			raws = append(raws, raw[begin:(i+1)])
			begin = i + 1
		}
	}
	raws = append(raws, raw[begin:])

	// raws = [ '{"cmd": "nikto -h www.maintstar.com -p 80"}', '{"cmd": "nikto -h www.maintstar.com -p 443"}' ]

	callmsg := "Calling:\n  " + strings.Join(raws, "\n  ")

	sendWebMessage("assistant", callmsg, w)
	*history = append(*history, ChatMessage{Role: "assistant", Content: callmsg})

	var args struct {
		Cmd string `json:"cmd"`
	}

	//rawsLength := len(raws)
	//var outputs []string

	for i := range raws {
		err := json.Unmarshal([]byte(raws[i]), &args)
		if err != nil {
			return "", []string{}, err
		}

		msg := fmt.Sprintf("Command %d Output: \n", i+1)
		output += msg
		sendWebMessage("tool", msg, w)

		// run the command
		tempOut, exitCode, err := runCommand(args.Cmd, w, i+1) // Calls sendWebMessage() within it to stream command output by line

		output += tempOut

		if err != nil {
			// outputs[i] = fmt.Sprintf("Error: %v", err)
			// break
			return "", []string{}, err // ---------------------------stops all the commands because if this one doesn't work then the next ones probably won't. Need to add which command failed
		}

		exitMsg := fmt.Sprintf("\nCommand %d Exit Code: %d", i+1, exitCode)

		if i < len(raws)-1 { // if it is not the iteration then make space
			exitMsg += "\n\n"
		}

		sendWebMessage("tool", exitMsg, w)
		output += exitMsg

		outputs = append(outputs, tempOut+exitMsg)
	}

	// output = "Command 1 Output: ... \n Exit Code: (Exit Code) \n Command 2 Output: ... \n Exit Code: (Exit Code)"

	return output, outputs, nil
}

func summarizeOut(output string, toolCall ToolCall, w http.ResponseWriter) string {

	// read system message and add to messages
	// add user message to messages. ""

	// Custom system message
	// Assistant message (with toolCall command and intent) (but without any content)
	// Tool message (command output)
	// User message "Summarize above cmd output keeping everything that may be useful based on the intent"

	//read system message
	systemmessagebyte, err := os.ReadFile("system_message.txt")
	systemmessage := string(systemmessagebyte)
	if err != nil {
		log.Fatal(err)
	}

	smessages := []ChatMessage{{Role: "system", Content: systemmessage}}

	// Add assistant message with toolCall containing cmd and intention
	// assisMessage := messages[len(messages)-1]
	// assisMessage.Content = ""
	// smessages = append(smessages, assisMessage)
	smessages = append(smessages, ChatMessage{Role: "assistant", Content: "", ToolCalls: []ToolCall{toolCall}})

	// Add tool message (command output)
	smessages = append(smessages, ChatMessage{Role: "tool", Content: output, ToolCallID: toolCall.ID})

	// Add user message ()
	// Please summarize all messages before but keep the progress and all the findings so we continue later
	smessages = append(smessages, ChatMessage{Role: "user", Content: "Please rewrite this output, summarizing it, only leaving important findings based on intention of command and penetration testing knowledge. Keep the labeling and exit codes for all commands"})

	//---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
	// Call the openAI API

	key := os.Getenv("OPENAI_API_KEY")
	ctx := context.TODO()

	// Tool definition
	tools := []ToolDef{}

	req := ChatRequest{
		Model:    "gpt-4o-mini", // same as openai.ChatModelGPT4oMini
		Messages: smessages,
		Tools:    tools,
		Stream:   true,
	}

	shortenedOut := ""

	prefix := "   ***Summarized ToolCall Output***\n\n"
	sendWebMessage("tool", prefix, w)
	shortenedOut += prefix

	chatResp, err := callOpenAIStream(ctx, key, req, w, "tool")
	shortenedOut += chatResp.Choices[0].Message.Content

	smessages = append(smessages, ChatMessage{Role: "assistant", Content: shortenedOut})

	number := 0

	fileName := "./chats/chat/smessages.json"

	for true {
		if number != 0 {
			fileName = fmt.Sprintf("./chats/chat/smessages%v.json", number)
		}
		// Check if the folder exists
		_, err := os.Stat(fileName)

		if err != nil && !os.IsNotExist(err) { // if err isn't "doesn't exist"
			// Error checking file
			log.Fatal(err)
		}

		// If file doesn't exist -> create it
		if os.IsNotExist(err) {
			// File doesn't exist
			updateMemory(&smessages, fileName)
			break
		}
		number++
	}

	return shortenedOut
}

func countTokens(message string) int {
	// allcontent := ""
	// for i := 0; i < len(messages); i++ { // traverse messages
	// 	allcontent += (messages[i].Content + " ")
	// }

	enc, err := tiktoken.EncodingForModel("gpt-4o-mini") // or your model
	if err != nil {
		log.Fatal(err)
	}

	// Not precise, now, openAI includes everything including role, content, toolCalls, and JSON structure. Later can add those or add a buffer (when precision required)
	tokens := len(enc.Encode(message, nil, nil))

	return tokens
}
