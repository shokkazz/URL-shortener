package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"shortener-service/internal/config"
	"strings"
	"time"
)

type Level int

const (
	Trace Level = iota
	Debug
	Info
	Warn
	Error
	Fatal
)

func (l Level) slogLevel() slog.Level {
	switch l {
	case Trace:
		return slog.LevelDebug - 4
	case Debug:
		return slog.LevelDebug
	case Info:
		return slog.LevelInfo
	case Warn:
		return slog.LevelWarn
	case Error:
		return slog.LevelError
	case Fatal:
		return slog.LevelError + 4
	default:
		return slog.LevelInfo
	}
}

func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return Trace, nil
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "warn", "warning":
		return Warn, nil
	case "error":
		return Error, nil
	case "fatal":
		return Fatal, nil
	default:
		return Info, fmt.Errorf("unknown log level %q", s)
	}
}

type Logger struct {
	base  *slog.Logger
	level *slog.LevelVar
}

func New(cfg config.Config) (*Logger, error) {
	level, err := ParseLevel(cfg.Logger.Level)
	if err != nil {
		return nil, err
	}

	host := cfg.Logger.HostName
	if host == "" {
		host, _ = os.Hostname()
	}

	pod := cfg.Logger.PodName

	levelVar := new(slog.LevelVar)
	levelVar.Set(level.slogLevel())

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelVar,
		ReplaceAttr: func(
			_ []string,
			a slog.Attr,
		) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				t := a.Value.Time().UTC().Truncate(time.Millisecond)
				return slog.String(
					"@timestamp",
					t.Format("2006-01-02T15:04:05.000Z"),
				)

			case slog.LevelKey:
				var logLevel slog.Level

				if value, ok := a.Value.Any().(slog.Level); ok {
					logLevel = value
				} else {
					return slog.String(
						"log.level",
						strings.ToUpper(a.Value.String()),
					)
				}

				switch logLevel {
				case slog.LevelDebug - 4:
					return slog.String("log.level", "TRACE")
				case slog.LevelDebug:
					return slog.String("log.level", "DEBUG")
				case slog.LevelInfo:
					return slog.String("log.level", "INFO")
				case slog.LevelWarn:
					return slog.String("log.level", "WARN")
				case slog.LevelError:
					return slog.String("log.level", "ERROR")
				case slog.LevelError + 4:
					return slog.String("log.level", "FATAL")
				default:
					return slog.String(
						"log.level",
						strings.ToUpper(logLevel.String()),
					)
				}

			case slog.MessageKey:
				return slog.String("message", a.Value.String())
			}

			return a
		},
	})

	var attrs []any

	if host != "" {
		attrs = append(attrs, slog.String("host.name", host))
	}

	if pod != "" {
		attrs = append(attrs, slog.String("k8s.pod.name", pod))
	}

	base := slog.New(handler).With(attrs...)

	return &Logger{
		base:  base,
		level: levelVar,
	}, nil
}

func (l *Logger) SetLevel(level Level) {
	l.level.Set(level.slogLevel())
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		base:  l.base.With(args...),
		level: l.level,
	}
}

func (l *Logger) log(ctx context.Context, level Level, msg string, args ...any) {
	l.base.Log(ctx, level.slogLevel(), msg, args...)
}

func (l *Logger) Trace(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Trace, msg, args...)
}

func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Debug, msg, args...)
}

func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Info, msg, args...)
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Warn, msg, args...)
}

func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Error, msg, args...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, args ...any) {
	l.log(ctx, Fatal, msg, args...)
}
