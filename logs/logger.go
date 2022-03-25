package logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	requestid "github.com/let-commerce/backend-common/request-id"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var (
	LogWriter   io.Writer
	ServiceName string
	Env         string
	ctx         *gin.Context
)

const defaultLogPath = "gin.log"

func SetRequestId(ginCtx *gin.Context) {
	ctx = ginCtx
}

func InitLogger(path string, env string, serviceName string) {
	// Setting Gin Logger
	if path == "" {
		path = defaultLogPath
	}
	f, err := os.Create(path)
	if err != nil {
		log.Panicf("can't open log file: gin.log, error: %s", err)
	}
	LogWriter = io.MultiWriter(os.Stdout, f)
	gin.DefaultWriter = LogWriter
	if env != "local" {
		gin.SetMode(gin.ReleaseMode)
	}
	Env = env
	ServiceName = serviceName
	log.SetReportCaller(true)
	log.SetOutput(LogWriter)
	if env == "prod" {
		log.SetFormatter(&JsonFormatter{TimestampFormat: "2006-01-02 15:04:05", LevelDesc: []string{"CRITICAL", "CRITICAL", "ERROR", "WARNING", "INFO", "DEBUG"}})
	} else {
		log.SetFormatter(&PlainFormatter{TimestampFormat: "2006-01-02 15:04:05", LevelDesc: []string{"PANIC", "FATAL", "ERROR", "WARN", "INFO", "DEBUG"}})
	}
}

type PlainFormatter struct {
	TimestampFormat string
	LevelDesc       []string
}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := fmt.Sprintf(entry.Time.Format(f.TimestampFormat))
	var requestId string
	if ctx != nil {
		requestId = requestid.GetRequestIDFromContext(ctx)
	}
	return []byte(fmt.Sprintf("[%s] [%s] - %s [%v:%v:%v - %v]\n", f.LevelDesc[entry.Level], timestamp, entry.Message, ServiceName, Env, requestId, Caller(entry.Caller))), nil
}

func Caller(f *runtime.Frame) string {
	p, _ := os.Getwd()
	fileName := strings.ReplaceAll(f.File, p, "")
	fileName = strings.ReplaceAll(fileName, "/go/pkg/mod/github.com/let-commerce/", "")
	return fmt.Sprintf("%s:%d", fileName, f.Line)
}

type JsonFormatter struct {
	TimestampFormat   string
	LevelDesc         []string
	PrettyPrint       bool
	DisableHTMLEscape bool
}

func (f *JsonFormatter) Format(entry *log.Entry) ([]byte, error) {
	result := map[string]interface{}{}

	result["timestamp"] = fmt.Sprintf(entry.Time.Format(f.TimestampFormat))
	if entry.HasCaller() {
		source := map[string]interface{}{}

		function, file, line := CallerWithFunc(entry.Caller)
		if function != "" {
			source["function"] = function
		}
		source["line"] = line
		source["file"] = file

		result["sourceLocation"] = source
	}
	result["level"] = entry.Level.String()
	result["severity"] = f.LevelDesc[entry.Level]
	result["serviceName"] = ServiceName
	result["env"] = Env

	requestId := ""
	if ctx != nil {
		requestId = requestid.GetRequestIDFromContext(ctx)
		result["request_id"] = requestId
		result["spanId"] = requestId

		httpRequest := map[string]interface{}{}
		httpRequest["requestMethod"] = ctx.Request.Method
		httpRequest["requestUrl"] = ctx.Request.RequestURI
		if ctx.Request.Response != nil {
			httpRequest["status"] = ctx.Request.Response.Status
		}
		httpRequest["remoteIp"] = ctx.Request.RemoteAddr
		result["httpRequest"] = httpRequest
	}
	result["message"] = fmt.Sprintf("%s [%v:%v:%v - %v]", entry.Message, ServiceName, Env, requestId, Caller(entry.Caller))

	b := &bytes.Buffer{}
	encoder := json.NewEncoder(b)
	encoder.SetEscapeHTML(!f.DisableHTMLEscape)
	if f.PrettyPrint {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(result); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}

	return b.Bytes(), nil
}

func CallerWithFunc(f *runtime.Frame) (string, string, string) {
	p, _ := os.Getwd()
	fileName := strings.ReplaceAll(f.File, p, "")
	fileName = strings.ReplaceAll(fileName, "/go/pkg/mod/github.com/let-commerce/", "")
	return f.Func.Name(), fileName, strconv.Itoa(f.Line)
}
