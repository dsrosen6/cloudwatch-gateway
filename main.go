package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/lambda"
)

var logger *slog.Logger

func init() {
	ctx := context.Background()

	h, err := newLogger(ctx, slog.LevelDebug)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

	logger = slog.New(h)
}

func main() {
	lambda.Start(handleRequest)
}
