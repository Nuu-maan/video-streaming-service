package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	logger zerolog.Logger
}

func New(environment, level string) *Logger {
	var output io.Writer = os.Stdout

	if environment == "development" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	logger := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	zerolog.SetGlobalLevel(logLevel)
	log.Logger = logger

	return &Logger{logger: logger}
}

func (l *Logger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	event := l.logger.Info()
	l.addFields(event, ctx, fields)
	event.Msg(msg)
}

func (l *Logger) Error(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	event := l.logger.Error()
	if err != nil {
		event = event.Err(err)
	}
	l.addFields(event, ctx, fields)
	event.Msg(msg)
}

func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	event := l.logger.Debug()
	l.addFields(event, ctx, fields)
	event.Msg(msg)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	event := l.logger.Warn()
	l.addFields(event, ctx, fields)
	event.Msg(msg)
}

func (l *Logger) Fatal(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	event := l.logger.Fatal()
	if err != nil {
		event = event.Err(err)
	}
	l.addFields(event, ctx, fields)
	event.Msg(msg)
}

func (l *Logger) addFields(event *zerolog.Event, ctx context.Context, fields map[string]interface{}) {
	if requestID := ctx.Value("request_id"); requestID != nil {
		event.Str("request_id", requestID.(string))
	}

	for key, value := range fields {
		event.Interface(key, value)
	}
}

func (l *Logger) GetZerolog() zerolog.Logger {
	return l.logger
}
