package auth

import "github.com/gin-gonic/gin"

func GetAuthenticatedConsumerId(ctx *gin.Context) uint {
	consumerId, exists := ctx.Get("AUTHENTICATED_CONSUMER_ID")
	if exists {
		return consumerId.(uint)
	}
	return 0
}

func GetIsAdmin(ctx *gin.Context) bool {
	isAdmin, exists := ctx.Get("IS_ADMIN")
	if exists {
		return isAdmin.(bool)
	}
	return false
}
