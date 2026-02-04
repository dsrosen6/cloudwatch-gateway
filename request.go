package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type attr struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type request struct {
	LogGroup      string `json:"log_group"`
	Level         string `json:"level,omitempty"`
	Message       string `json:"message"`
	Attributes    []attr `json:"attributes,omitempty"`
	RetentionDays *int32 `json:"retention_days,omitempty"`
}

type response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func handleRequest(ctx context.Context, request request) (response, error) {
	if request.LogGroup == "" {
		return response{
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

	record := slog.NewRecord(time.Now(), level, request.Message, 0)
	record.AddAttrs(attrs...)

	if err := logger.Handler().Handle(ctx, record); err != nil {
		return response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return response{
		Success: true,
		Message: "Log written successfully",
	}, nil
}
