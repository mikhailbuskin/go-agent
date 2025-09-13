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

type APIChatResponse struct {
	Content string `json:"content"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) > 1 && os.Args[1] == "cli" {
		agentConsole()
	} else {
		fmt.Println("Listening to port 8080")
		mux := http.NewServeMux()
		mux.HandleFunc("/api/chat", chatHandler)
		http.ListenAndServe(":8080", enableCORS(mux))
	}
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	var req APIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	responce := agentWeb(req.Message)
	// Call your AI agent logic here
	//aiResponse := "Echo: " + req.Message
	aiResponse := responce

	res := APIChatResponse{Content: aiResponse}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
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
