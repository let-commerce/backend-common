package auth

import "github.com/gin-gonic/gin"

func GetAuthenticatedConsumerId(ctx *gin.Context) uint {
	consumerId, exists := ctx.Get("AUTHENTICATED_CONSUMER_ID")
	if exists {
		return consumerId.(uint)
	}
	return 0
}
