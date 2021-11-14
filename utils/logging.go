package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var singletonLog *logrus.Logger
var once sync.Once

// SetLogger creates the logger object
func SetLogger() *logrus.Logger {
	once.Do(func() {
		if singletonLog == nil {
			file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
			singletonLog = logrus.New()
			singletonLog.Level = logrus.InfoLevel
			singletonLog.SetReportCaller(true)
			formatter := &logrus.TextFormatter{
				FullTimestamp:          true,
				TimestampFormat:        "02-01-2006 15:04:05",
				DisableColors:          true,
				DisableLevelTruncation: true,
				CallerPrettyfier: func(f *runtime.Frame) (string, string) {
					return "", fmt.Sprintf("%s:%d", formatFilePath(f.File), f.Line)
				},
			}
			singletonLog.SetFormatter(formatter)
			if err != nil {
				panic(err) // Cannot open log file. Logging to stderr
			} else {
				singletonLog.SetOutput(file)
			}
		}
	})
	return singletonLog
}

// GetLogger returns the logger object
func GetLogger() *logrus.Logger {
	return SetLogger()
}

func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}
