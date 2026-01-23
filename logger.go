package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type (
	cloudWatchClient interface {
		CreateLogGroup(ctx context.Context, params *cloudwatchlogs.CreateLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error)
		CreateLogStream(ctx context.Context, params *cloudwatchlogs.CreateLogStreamInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogStreamOutput, error)
		PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error)
		PutRetentionPolicy(ctx context.Context, params *cloudwatchlogs.PutRetentionPolicyInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error)
	}

	handler struct {
		client         cloudWatchClient
		sequenceToken  *string
		mu             sync.Mutex
		attrs          []slog.Attr
		groups         []string
		logLevel       slog.Level
		createdGroups  map[string]bool
		createdStreams map[string]string // maps original stream name to timestamped version
	}

	contextKey string
)

const (
	logGroupKey      contextKey = "cloudwatch_log_group"
	retentionDaysKey contextKey = "cloudwatch_retention_days"
)

func newLogger(ctx context.Context, level slog.Level) (*handler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(cfg)

	return &handler{
		client:         client,
		logLevel:       level,
		createdGroups:  make(map[string]bool),
		createdStreams: make(map[string]string),
	}, nil
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.logLevel
}

func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	logEntry := map[string]any{
		"time":    record.Time.Format(time.RFC3339),
		"level":   record.Level.String(),
		"message": record.Message,
	}

	for _, attr := range h.attrs {
		addAttrToMap(logEntry, attr)
	}

	record.Attrs(func(attr slog.Attr) bool {
		addAttrToMap(logEntry, attr)
		return true
	})

	var groupName string
	if ctxGroup := ctx.Value(logGroupKey); ctxGroup != nil {
		if g, ok := ctxGroup.(string); ok && g != "" {
			groupName = g
		}
	}

	if groupName == "" {
		return fmt.Errorf("log_group is required in context")
	}

	// Always use "log-stream" as the base stream name
	streamName := "log-stream"

	if len(h.groups) > 0 {
		logEntry["groups"] = h.groups
	}

	message, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if err := h.ensureLogGroup(ctx, groupName); err != nil {
		return fmt.Errorf("ensuring log group: %w", err)
	}

	actualStreamName, err := h.ensureLogStream(ctx, groupName, streamName)
	if err != nil {
		return fmt.Errorf("ensuring log stream: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(actualStreamName),
		LogEvents: []types.InputLogEvent{
			{
				Message:   aws.String(string(message)),
				Timestamp: aws.Int64(record.Time.UnixMilli()),
			},
		},
	}

	result, err := h.client.PutLogEvents(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put log events: %w", err)
	}

	h.sequenceToken = result.NextSequenceToken
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &handler{
		client:         h.client,
		sequenceToken:  h.sequenceToken,
		attrs:          newAttrs,
		groups:         h.groups,
		logLevel:       h.logLevel,
		createdGroups:  h.createdGroups,
		createdStreams: h.createdStreams,
	}
}

func (h *handler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &handler{
		client:         h.client,
		sequenceToken:  h.sequenceToken,
		attrs:          h.attrs,
		groups:         newGroups,
		logLevel:       h.logLevel,
		createdGroups:  h.createdGroups,
		createdStreams: h.createdStreams,
	}
}

func (h *handler) ensureLogGroup(ctx context.Context, groupName string) error {
	if h.createdGroups[groupName] {
		return nil
	}

	_, err := h.client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(groupName),
	})

	h.createdGroups[groupName] = true

	if err != nil {
		// log group probably already exists
		return nil
	}

	if retentionDays := ctx.Value(retentionDaysKey); retentionDays != nil {
		if days, ok := retentionDays.(int32); ok {
			_, _ = h.client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
				LogGroupName:    aws.String(groupName),
				RetentionInDays: aws.Int32(days),
			})
		}
	}

	return nil
}

func (h *handler) ensureLogStream(ctx context.Context, groupName, streamName string) (string, error) {
	key := fmt.Sprintf("%s/%s", groupName, streamName)

	if timestampedName, exists := h.createdStreams[key]; exists {
		return timestampedName, nil
	}

	// Append timestamp to stream name to make it unique per Lambda instance
	timestampedName := fmt.Sprintf("%s-%s", streamName, time.Now().Format("2006-01-02-15-04-05"))

	_, err := h.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(timestampedName),
	})

	h.createdStreams[key] = timestampedName

	if err != nil {
		// Log stream may already exist
		return timestampedName, nil
	}

	return timestampedName, nil
}

func addAttrToMap(m map[string]any, attr slog.Attr) {
	if attr.Value.Kind() == slog.KindGroup {
		// Handle group attributes by creating a nested map
		groupMap := make(map[string]any)
		for _, groupAttr := range attr.Value.Group() {
			addAttrToMap(groupMap, groupAttr)
		}
		m[attr.Key] = groupMap
	} else {
		val := attr.Value.Any()
		if err, ok := val.(error); ok {
			m[attr.Key] = err.Error()
		} else {
			m[attr.Key] = val
		}
		m[attr.Key] = attr.Value.Any()
	}
}
