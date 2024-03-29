package logger

import (
	"context"
	"log/slog"
	"os"
	"oteltail/pkg/version"
	"runtime/debug"
)

// Default logger of the system.
var logger *slog.Logger

type ctxKey string

const (
	slogFields ctxKey = "slog_fields"
)

type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.Handler.Handle(ctx, r)
}

func AppendCtx(parent context.Context, attr slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(slogFields).([]slog.Attr); ok {
		v = append(v, attr)
		return context.WithValue(parent, slogFields, v)
	}

	v := []slog.Attr{}
	v = append(v, attr)
	return context.WithValue(parent, slogFields, v)
}

func GetLogger(ctx context.Context) *slog.Logger {

	// return pre-inited logger
	if logger != nil {
		return logger
	}

	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = "INFO"
	}

	buildInfo, _ := debug.ReadBuildInfo()

	//
	var level slog.Level
	// display source code lines
	var addSource = false

	// init new logger
	switch logLevel {
	case "ERR", "ERROR":
		level = slog.LevelError
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "DEBUG":
		level = slog.LevelDebug
		addSource = true
	case "INFO":
		level = slog.LevelInfo
	default:
		level = slog.LevelInfo
	}

	handler := &ContextHandler{slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: addSource, Level: level}).
		WithAttrs([]slog.Attr{
			slog.String("bv", version.Version),
			slog.String("bh", version.BuildHash),
			slog.String("bd", version.BuildDate),
			slog.String("gv", buildInfo.GoVersion),
		})}

	logger := slog.New(handler)

	return logger
}
