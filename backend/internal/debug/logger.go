package debug

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelOff LogLevel = iota
	LevelBasic
	LevelVerbose
	LevelTrace
)

var (
	debugLevel LogLevel
	logger     *log.Logger
)

func init() {
	logger = log.New(os.Stderr, "", 0)
	updateDebugLevel()
}

func updateDebugLevel() {
	debugEnv := strings.ToLower(os.Getenv("MCP_DEBUG"))
	switch debugEnv {
	case "true", "1", "basic":
		debugLevel = LevelBasic
	case "verbose":
		debugLevel = LevelVerbose
	case "trace":
		debugLevel = LevelTrace
	default:
		debugLevel = LevelOff
	}
}

func IsEnabled() bool {
	return debugLevel > LevelOff
}

func IsVerbose() bool {
	return debugLevel >= LevelVerbose
}

func IsTrace() bool {
	return debugLevel >= LevelTrace
}

func formatMessage(level string, format string, args ...interface{}) string {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		file = filepath.Base(file)
	} else {
		file = "???"
		line = 0
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s %s:%d %s", level, timestamp, file, line, message)
}

func Log(format string, args ...interface{}) {
	if debugLevel >= LevelBasic {
		logger.Println(formatMessage("DEBUG", format, args...))
	}
}

func Logf(format string, args ...interface{}) {
	Log(format, args...)
}

func Verbose(format string, args ...interface{}) {
	if debugLevel >= LevelVerbose {
		logger.Println(formatMessage("VERBOSE", format, args...))
	}
}

func Trace(format string, args ...interface{}) {
	if debugLevel >= LevelTrace {
		logger.Println(formatMessage("TRACE", format, args...))
	}
}

func SQL(query string, args ...interface{}) {
	if debugLevel >= LevelVerbose {
		logger.Println(formatMessage("SQL", "Query: %s | Args: %v", query, args))
	}
}

func Request(method string, endpoint string, body interface{}) {
	if debugLevel >= LevelBasic {
		if body != nil {
			logger.Println(formatMessage("REQUEST", "%s %s | Body: %v", method, endpoint, body))
		} else {
			logger.Println(formatMessage("REQUEST", "%s %s", method, endpoint))
		}
	}
}

func Response(status int, body interface{}) {
	if debugLevel >= LevelBasic {
		if debugLevel >= LevelVerbose && body != nil {
			logger.Println(formatMessage("RESPONSE", "Status: %d | Body: %v", status, body))
		} else {
			logger.Println(formatMessage("RESPONSE", "Status: %d", status))
		}
	}
}

func Error(format string, args ...interface{}) {
	if debugLevel >= LevelBasic {
		logger.Println(formatMessage("ERROR", format, args...))
	}
}