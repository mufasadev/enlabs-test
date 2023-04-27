package log

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"sync"
	"time"
)

var logger zerolog.Logger
var once sync.Once

type LoggerOption func(*LoggerConfig)

type LoggerConfig struct {
	fileName string
	console  bool
	logLevel int
}

func WithFileLogger(fileName string) LoggerOption {
	return func(l *LoggerConfig) {
		l.fileName = fileName
	}
}

func WithConsoleLogger() LoggerOption {
	return func(l *LoggerConfig) {
		l.console = true
	}
}

func WithLogLevel(logLevel int) LoggerOption {
	return func(l *LoggerConfig) {
		l.logLevel = logLevel
	}
}

func Init(serviceName string, opts ...LoggerOption) {
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano
		l := &LoggerConfig{}
		l.logLevel = int(zerolog.InfoLevel)

		for _, opt := range opts {
			opt(l)
		}

		output := make([]io.Writer, 0, 2)
		defaultOutput := os.Stdout
		if l.console {
			consoleOutput := zerolog.ConsoleWriter{
				Out:        defaultOutput,
				TimeFormat: time.RFC3339,
			}
			output = append(output, consoleOutput)
		}
		if l.fileName != "" {
			fileOutput := &lumberjack.Logger{
				Filename:   l.fileName,
				MaxSize:    5,
				MaxBackups: 10,
				MaxAge:     14,
				Compress:   true,
			}
			output = append(output, fileOutput)
		}

		if len(output) == 0 {
			output = append(output, defaultOutput)
		}

		multiWriter := zerolog.MultiLevelWriter(output...)

		logger = zerolog.New(multiWriter).
			Level(zerolog.Level(l.logLevel)).
			With().
			Timestamp().
			Str("service", serviceName).
			Logger()
	})
}

func GetLogger() zerolog.Logger {
	return logger
}
