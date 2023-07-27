package utils

import (
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func InitLogging(debug bool) {
	logger = logrus.New()

	if debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
}

func LogError(message string) {
	logger.Error(message)
}

func LogInfo(message string) {
	logger.Info(message)
}

func LogDebug(message string) {
	logger.Debug(message)
}
