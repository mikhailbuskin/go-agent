package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type APIChatRequest struct {
	Message string `json:"message"`
}

type APIChatMemoryResponse struct {
	Messages []ChatMessage `json:"messages"`
}

type APIChatResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Summarization test
	//-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------

	// // 1. deser existing messages log
	// // combine with summarization system message + messages + message("summarize please")
	// // call api, check results.

	// fileNameM := "./chat7 (test2)/messages.json"

	// //read starter messages
	// //startmessagebyte, err := os.ReadFile("summarization_system_message.txt")
	// // startmessagebyte, err := os.ReadFile("system_message.txt")
	// // startmessagestr := string(startmessagebyte)
	// // if err != nil {
	// // 	log.Fatal(err)
	// // }

	// //messages := []ChatMessage{{Role: "system", Content: startmessagestr}}

	// // Read file
	// contentsM, err := os.ReadFile(fileNameM) // contentsM is a JSON string in bytes
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// messages := []ChatMessage{}
	// // Load file into messages
	// err = json.Unmarshal(contentsM, &messages)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// //messages = append(messages, temp...)
	// messages = append(messages, ChatMessage{Role: "user", Content: "Please summarize all messages before but keep the progress and all the findings so we continue later"})

	// //--------------------------------------------------------------------------------

	// key := os.Getenv("OPENAI_API_KEY")
	// ctx := context.TODO()

	// // Tool definition
	// tools := []ToolDef{
	// 	{
	// 		Type: "function",
	// 		Function: FunctionDef{
	// 			Name:        TOOL_RUN_COMMAND,
	// 			Description: "Run a command in Kali Linux environment",
	// 			Parameters: map[string]interface{}{
	// 				"type": "object",
	// 				"properties": map[string]interface{}{
	// 					"cmd": map[string]string{
	// 						"type": "string",
	// 					},
	// 				},
	// 				"required": []string{"cmd"},
	// 			},
	// 		},
	// 	},
	// }

	// req := ChatRequest{
	// 	Model:    "gpt-4o-mini", // same as openai.ChatModelGPT4oMini
	// 	Messages: messages,
	// 	Tools:    tools,
	// 	Stream:   false,
	// }

	// chatResp, err := callOpenAI(ctx, key, req)

	// fmt.Println(chatResp.Choices[0].Message.Content)

	// data, _ := json.MarshalIndent(chatResp, "", " ") // json.MarshalIndent (and json.Marshal) can take either a value or a pointer to a value/structure. It doesn’t require the value specifically.
	// err = os.WriteFile("summarization_output", data, 0644)
	// if err != nil {
	// 	log.Fatalf("Error writing to file: %v", err)
	// }

	// return

	//-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------

	// args is what you put after the run command. So in "go run . cli something" the argument[1] would be "cli" and argument[2] would be "something"
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		//agentConsole()
	} else {
		fmt.Println("Listening to port 8080")
		mux := http.NewServeMux()
		mux.HandleFunc("/api/chat", chatHandler)
		mux.HandleFunc("/api/memory", sendMemory)
		http.ListenAndServe(":8080", enableCORS(mux))
	}
}

func sendMemory(w http.ResponseWriter, r *http.Request) {
	fileName := "./chat/history.json"

	var messages []ChatMessage
	//var filtered []ChatMessage

	// Check if file exists.
	_, err := os.Stat(fileName)
	if err != nil {
		// File doesn't exit
		messages = []ChatMessage{}
		// Return empty messages

	} else {
		// File exists

		// Read file
		byteMessages, err := os.ReadFile(fileName) // Read File gives you the contents in []byte data type
		json.Unmarshal(byteMessages, &messages)

		if err != nil {
			log.Fatal(err)
		}

		// for _, msg := range messages {
		// 	if msg.Role == "assistant" || msg.Role == "user" {
		// 		if msg.Content != "" {
		// 			filtered = append(filtered, msg)
		// 		}
		// 	}
		// }
	}

	res := APIChatMemoryResponse{Messages: messages}
	bytes, err := json.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(res)
	w.Write(bytes)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	var req APIChatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	agentWeb(req.Message, w) // Calls sendWebMessage() within agentWeb() to stream the info.

	// Below code was used previously to print responce all at once:

	// Call your AI agent logic here
	//aiResponse := "Echo: " + req.Message
	//aiResponse := responce

	// res := APIChatResponse{Content: response}
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(res)
}

func sendWebMessage(role string, content string, w http.ResponseWriter) error {
	msg := APIChatResponse{Role: role, Content: content} // send along with the role so web knows in which color to display it
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write JSON + newline
	// if _, err := w.Write(append(data, '\n')); err != nil {
	// 	return err
	// }
	if _, err := fmt.Fprintf(w, "data: %s\n", data); err != nil {
		return err
	}

	// Force flush so client receives immediately
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Allow common headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// Allow common methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
