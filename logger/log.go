package logger

import "github.com/sirupsen/logrus"

var Logger *logrus.Logger

func init() {
	Logger = logrus.New()
	Logger.SetReportCaller(true)
}
