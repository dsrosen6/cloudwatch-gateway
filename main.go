package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

type Attribute struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type LogRequest struct {
	LogGroup      string      `json:"log_group"`
	Level         string      `json:"level,omitempty"`
	Message       string      `json:"message"`
	Attributes    []Attribute `json:"attributes,omitempty"`
	RetentionDays *int32      `json:"retention_days,omitempty"`
}

type LogResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

var logger *slog.Logger

func init() {
	ctx := context.Background()

	h, err := newLogger(ctx, slog.LevelDebug)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}

	logger = slog.New(h)
}

func handleRequest(ctx context.Context, request LogRequest) (LogResponse, error) {
	if request.LogGroup == "" {
		return LogResponse{
			Success: false,
			Error:   "log_group is required",
		}, fmt.Errorf("log_group is required")
	}

	level := slog.LevelInfo
	switch strings.ToLower(request.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	ctx = context.WithValue(ctx, logGroupKey, request.LogGroup)
	if request.RetentionDays != nil {
		ctx = context.WithValue(ctx, retentionDaysKey, *request.RetentionDays)
	}

	attrs := make([]slog.Attr, 0, len(request.Attributes))
	for _, attr := range request.Attributes {
		attrs = append(attrs, slog.Any(attr.Key, attr.Value))
	}

	logger.LogAttrs(ctx, level, request.Message, attrs...)

	return LogResponse{
		Success: true,
		Message: "Log written successfully",
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
