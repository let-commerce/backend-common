// Package middlewares contains gin middlewares
// Usage: router.Use(middlewares.Connect)
package middlewares

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/let-commerce/backend-common/ginutils"
	"github.com/let-commerce/backend-common/logs"
	"github.com/let-commerce/backend-common/response"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func LogErrorResponse(ctx *gin.Context) {
	blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
	ctx.Writer = blw
	ctx.Next()
	statusCode := ctx.Writer.Status()
	if statusCode >= 400 {
		// Record the response body if there was an error
		log.Warnf("Returning error status code [%v] for request: [%v] %v - Response Body is: %v.", statusCode, ctx.Request.Method, ctx.Request.RequestURI, blw.body.String())
	}
}

func InitGinCtx(ctx *gin.Context) {
	ginutils.Init(ctx)
	response.Init(ctx)
	logs.SetRequestId(ctx)
	ctx.Next()
}

func LogAllRequests(ctx *gin.Context) {
	buf, _ := ioutil.ReadAll(ctx.Request.Body)
	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf)) //We have to create a new Buffer, because rdr1 will be read.

	if !strings.Contains(ctx.Request.RequestURI, "swagger") {
		log.Infof("Start handling reuqest for URI: [%v] %v - Params: %v, Body: [%+v].", ctx.Request.Method, ctx.Request.RequestURI, ctx.Params, readBody(rdr1)) // Print request body
	}
	ctx.Request.Body = rdr2
	ctx.Next()
}

func LogAllResponses(ctx *gin.Context) {
	blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
	ctx.Writer = blw
	ctx.Next()
	statusCode := ctx.Writer.Status()
	if !strings.Contains(ctx.Request.RequestURI, "swagger") {
		if statusCode >= 400 && statusCode != 401 {
			log.Errorf("Finished handling request for URI: [%v] %v - Response is: [%v] %v.", ctx.Request.Method, ctx.Request.RequestURI, statusCode, blw.body.String())
		} else if statusCode == 401 {
			log.Warnf("Finished handling request for URI: [%v] %v - Response is: [%v] %v.", ctx.Request.Method, ctx.Request.RequestURI, statusCode, blw.body.String())
		} else {
			log.Infof("Finished handling request for URI: [%v] %v - Response is: [%v] %v.", ctx.Request.Method, ctx.Request.RequestURI, statusCode, blw.body.String())
		}
	}
}

func readBody(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	s := buf.String()
	return s
}

func RecoveryHandler(c *gin.Context, err interface{}) {
	stack := stack(3)
	log.Errorf("Got panic while handling [%v] %v: %+v, stack: %s", c.Request.Method, c.Request.RequestURI, err, stack)

	c.JSON(http.StatusInternalServerError, response.ErrorResponse{Message: "got panic", Error: fmt.Sprintf("%v", err)})
}

// stack returns a nicely formatted stack frame, skipping skip frames.
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return []byte("???")
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return []byte("???")
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := bytes.LastIndex(name, []byte("/")); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := bytes.Index(name, []byte(".")); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, []byte("·"), []byte("."), -1)
	return name
}
