package requestid

import (
	"encoding/base64"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	xRequestIDKey = "X-Request-ID"
)

// generator a function type that returns string.
type generator func() string

var (
	random = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
)

func uuid(len int) string {
	bytes := make([]byte, len)
	random.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)[:len]
}

//RequestID is a middleware that injects a 'RequestID' into the context and header of each request.
func RequestID(ctx *gin.Context) {
	xRequestID := uuid(8)

	ctx.Set(xRequestIDKey, xRequestID)
	ctx.Next()
}

// GetRequestIDFromContext returns 'RequestID' from the given context if present.
func GetRequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(xRequestIDKey); ok {
		if requestID, ok := v.(string); ok {
			return requestID
		}
	}
	return ""
}
