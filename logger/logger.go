package logger

import (
	"os"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type AgentLogger struct {
	infoLogger  logr.Logger
	errorLogger logr.Logger
}

func NewAgentLogger() logr.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	return AgentLogger{
		infoLogger:  ctrlzap.New(ctrlzap.WriteTo(os.Stdout), ctrlzap.Encoder(zapcore.NewJSONEncoder(config))),
		errorLogger: ctrlzap.New(ctrlzap.WriteTo(os.Stderr), ctrlzap.Encoder(zapcore.NewJSONEncoder(config))),
	}
}

func (log AgentLogger) Info(msg string, keysAndValues ...interface{}) {
	log.infoLogger.Info(msg, keysAndValues...)
}

func (log AgentLogger) Enabled() bool {
	return log.infoLogger.Enabled()
}

func (log AgentLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.errorLogger.Error(err, msg, keysAndValues...)
}

func (log AgentLogger) V(level int) logr.InfoLogger {
	return log.errorLogger.V(level)
}

func (log AgentLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return AgentLogger{
		infoLogger:  log.infoLogger.WithValues(keysAndValues...),
		errorLogger: log.errorLogger.WithValues(keysAndValues...),
	}
}

func (log AgentLogger) WithName(name string) logr.Logger {
	return AgentLogger{
		infoLogger:  log.infoLogger.WithName(name),
		errorLogger: log.errorLogger.WithName(name),
	}
}
