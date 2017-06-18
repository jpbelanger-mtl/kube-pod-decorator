package logger

import (
	"os"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("default")

func GetLogger() *logging.Logger {
	return log
}

func InitLogger(logLevel string) {
	var activeLogLevel, _ = logging.LogLevel(logLevel)

	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`))
	backend1Leveled := logging.AddModuleLevel(backend1Formatter)
	backend1Leveled.SetLevel(activeLogLevel, "default")
	logging.SetBackend(backend1Leveled)
}
