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
	fileName := "messages.json"

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
		messages = messages[1:]

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

	agentWeb(req.Message, w)
	// Call your AI agent logic here
	//aiResponse := "Echo: " + req.Message
	//aiResponse := responce

	// res := APIChatResponse{Content: response}
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(res)
}

func sendWebMessage(role string, content string, w http.ResponseWriter) error {
	// res := APIChatResponse{Content: response}
	// w.Header().Set("Content-Type", "application/json")
	// w.Header().Set("Cache-Control", "no-cache")
	// w.Header().Set("Connection", "keep-alive")
	// json.NewEncoder(w).Encode(res)

	msg := APIChatResponse{Role: role, Content: content}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write JSON + newline
	if _, err := w.Write(append(data, '\n')); err != nil {
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
