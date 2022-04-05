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

func GetAuthenticatedUid(ctx *gin.Context) string {
	uid, exists := ctx.Get("FIREBASE_USER_UID")
	if exists {
		return uid.(string)
	}
	return ""
}

func GetIsGuest(ctx *gin.Context) bool {
	isGuest, exists := ctx.Get("IS_GUEST")
	if exists {
		return isGuest.(bool)
	}
	return true
}
