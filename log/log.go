package log

import (
	"log"
	"os"

	"ghscraper.htm/system"
)

var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
	Fatal *log.Logger
)

func InitLogger() {
	var logFile *os.File
	var err error

	if system.IsDev() {
		logFile = os.Stderr
	} else {
		logFile, err = os.OpenFile(system.Cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal("Failed to set up logger")
		}
	}

	Info = log.New(logFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(logFile, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(logFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	Fatal = log.New(logFile, "[FATAL] ", log.Ldate|log.Ltime|log.Lshortfile)

	return
}
