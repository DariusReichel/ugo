package logger

import (
	"bytes"
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var logger *logrus.Logger
var logOnce sync.Once
var writer *rotatelogs.RotateLogs

func init() {
	logger = logrus.New()
}

func InitLogOnce(logPath, projectName string) {
	logOnce.Do(func() {
		initLogs(logPath, projectName)
	})
}
func initLogs(logPath, projectName string) {
	logWriter, err := rotatelogs.New(
		logPath+"/"+projectName+"_%Y%m%d.log",
		rotatelogs.WithLinkName(logPath+"/"+projectName+".log"), // 创建一个符号链接指向最新的日志文件
		rotatelogs.WithMaxAge(7*24*time.Hour),                   // 保留7天的日志文件
		rotatelogs.WithRotationTime(24*time.Hour),               // 每24小时切割一次日志文件
	)
	if err != nil {
		log.Fatal("Failed to create log writer:", err)
	}
	// 设置 logrus 的输出为 Rotatelogs 的日志文件写入器
	logger.SetOutput(io.MultiWriter(os.Stdout, logWriter))
}

// 日志自定义格式
type LogFormatter struct {
	ginRe *regexp.Regexp
}

func (s *LogFormatter) Init() {
	s.ginRe = regexp.MustCompile("(?m)[\r\n]+^.*gin-gonic.*$")
}

func LogStack() string {
	pc := make([]uintptr, 10)
	n := runtime.Callers(9, pc)
	frames := runtime.CallersFrames(pc[:n])

	var frame runtime.Frame
	more := n > 0
	output := ""
	for more {
		frame, more = frames.Next()
		output = output + fmt.Sprintf("%s:%d \n %s\n", frame.File, frame.Line, frame.Function)
	}
	return output
}

func (s *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("[2006-01-02|15:04:05.000]")
	var file string
	var l int
	if entry.Caller != nil {
		file = filepath.Base(entry.Caller.File)
		l = entry.Caller.Line
	}

	msg := fmt.Sprintf("%s[%s:%d][GID:%d][%s]: %s\n", timestamp, file, l, getGID(), strings.ToUpper(entry.Level.String()), entry.Message)
	if entry.Level <= logrus.ErrorLevel {
		stackInfo := LogStack()

		stackInfo = s.ginRe.ReplaceAllString(stackInfo, "")

		msg = msg + stackInfo
	}

	return []byte(msg), nil
}

// 获取当前协程ID
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// SetLogFile sets the log file path
func SetNewLogFile(logPath, projectName string) {
	// 关闭已有的日志文件
	Close()
	initLogs(logPath, projectName)
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Close closes the log file
func Close() {
	if writer != nil {
		writer.Close()
	}
}
