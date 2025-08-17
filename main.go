package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	key := os.Getenv("OPENAI_API_KEY")

	client := openai.NewClient(
		option.WithAPIKey(key), // defaults to os.LookupEnv("OPENAI_API_KEY")
	)

	ctx := context.TODO()

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("Say that this is a test"),
	}

	model := openai.ChatModelGPT4o

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
	})
	if err != nil {
		panic(err.Error())
	}
	println(chatCompletion.Choices[0].Message.Content)
}
