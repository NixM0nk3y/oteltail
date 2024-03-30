package logger

import (
	"context"
	"log/slog"
	"os"
	"oteltail/pkg/version"
	"path/filepath"
	"runtime/debug"

	"github.com/mdobak/go-xerrors"
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

type stackFrame struct {
	Func   string `json:"func"`
	Source string `json:"source"`
	Line   int    `json:"line"`
}

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		switch v := a.Value.Any().(type) {
		case error:
			a.Value = fmtErr(v)
		}
	}

	return a
}

// marshalStack extracts stack frames from the error
func marshalStack(err error) []stackFrame {
	trace := xerrors.StackTrace(err)

	if len(trace) == 0 {
		return nil
	}

	frames := trace.Frames()

	s := make([]stackFrame, len(frames))

	for i, v := range frames {
		f := stackFrame{
			Source: filepath.Join(
				filepath.Base(filepath.Dir(v.File)),
				filepath.Base(v.File),
			),
			Func: filepath.Base(v.Function),
			Line: v.Line,
		}

		s[i] = f
	}

	return s
}

// fmtErr returns a slog.Value with keys `msg` and `trace`. If the error
// does not implement interface { StackTrace() errors.StackTrace }, the `trace`
// key is omitted.
func fmtErr(err error) slog.Value {
	var groupValues []slog.Attr

	groupValues = append(groupValues, slog.String("msg", err.Error()))

	frames := marshalStack(err)

	if frames != nil {
		groupValues = append(groupValues,
			slog.Any("trace", frames),
		)
	}

	return slog.GroupValue(groupValues...)
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

	handler := &ContextHandler{slog.NewJSONHandler(
		os.Stderr,
		&slog.HandlerOptions{
			AddSource:   addSource,
			Level:       level,
			ReplaceAttr: replaceAttr,
		}).
		WithAttrs([]slog.Attr{
			slog.String("bv", version.Version),
			slog.String("bh", version.BuildHash),
			slog.String("bd", version.BuildDate),
			slog.String("gv", buildInfo.GoVersion),
		})}

	logger := slog.New(handler)

	return logger
}
