// Package middlewares contains gin middlewares
// Usage: router.Use(middlewares.Connect)
package middlewares

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/let-commerce/backend-common/env"
	"github.com/let-commerce/backend-common/logs"
	"github.com/let-commerce/backend-common/response"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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
	logs.SetRequestId(ctx)
	ctx.Next()
}

func LogAllRequests(ctx *gin.Context) {
	buf, _ := ioutil.ReadAll(ctx.Request.Body)
	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf)) //We have to create a new Buffer, because rdr1 will be read.

	if !strings.Contains(ctx.Request.RequestURI, "swagger") {
		body := readBody(rdr1)
		if body != "" {
			log.Infof("Start handling reuqest for URI: [%v] %v - Params: %v, Body: [%+v].", ctx.Request.Method, ctx.Request.RequestURI, ctx.Params, body) // Print request body
		} else {
			log.Infof("Start handling reuqest for URI: [%v] %v - Params: %v.", ctx.Request.Method, ctx.Request.RequestURI, ctx.Params) // Print request body
		}
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
		if statusCode >= 402 {
			log.Errorf("Finished handling request for URI: [%v] %v - Response is: [%v] %v.", ctx.Request.Method, ctx.Request.RequestURI, statusCode, blw.body.String())
		} else if statusCode == 400 || statusCode == 401 {
			log.Warnf("Finished handling request for URI: [%v] %v - Response is: [%v] %v.", ctx.Request.Method, ctx.Request.RequestURI, statusCode, blw.body.String())
		} else {
			if ctx.Request.Method == "GET" { // In order to prevent huge log files, logging only status for GET requests, if there was no error.
				log.Infof("Finished handling request for URI: [%v] %v - Response code is: [%v].", ctx.Request.Method, ctx.Request.RequestURI, statusCode)
			} else {
				log.Infof("Finished handling request for URI: [%v] %v - Response is: [%v] %v.", ctx.Request.Method, ctx.Request.RequestURI, statusCode, blw.body.String())
			}
		}
	}
}

func readBody(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	s := buf.String()
	return s
}

func RecoveryHandler(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			goErr := errors.Wrap(err, 3)
			log.Errorf("Got panic while handling [%v] %v: %+v, Stack:\n(Service: %v)\n%s", c.Request.Method, c.Request.RequestURI, err, env.GetEnvVar("SERVICE_NAME"), Caller(goErr.StackFrames(), 0))

			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Message: "got panic", Error: fmt.Sprintf("%v", err)})
		}
	}()
	c.Next() // execute all the handlers
}

func Caller(stack []errors.StackFrame, maxDepth int) string {
	result := ""
	p, _ := os.Getwd()

	for i, stackFrame := range stack {
		if maxDepth == 0 || i < maxDepth {
			fileName := strings.ReplaceAll(stackFrame.File, p, "")
			fileName = strings.ReplaceAll(fileName, "go/pkg/mod/github.com/let-commerce/", "")
			fileName = strings.ReplaceAll(fileName, "go/pkg/mod/github.com/gin-gonic/", "")
			result += fmt.Sprintf("%v:%v\n", fileName, stackFrame.LineNumber)
		}
	}
	return result
}
