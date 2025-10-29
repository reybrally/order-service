package logging

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger *logrus.Logger

func InitLogger() {
	Logger = logrus.New()
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(logrus.InfoLevel)
}

func LogInfo(message string, fields logrus.Fields) {
	Logger.WithFields(fields).Info(message)
}

func LogError(message string, err error, fields logrus.Fields) {
	fields["error"] = err.Error()
	Logger.WithFields(fields).Error(message)
}

func LogDebug(message string, fields logrus.Fields) {
	Logger.WithFields(fields).Debug(message)
}
