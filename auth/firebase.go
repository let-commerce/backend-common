package auth

import (
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/let-commerce/backend-common/env"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"gorm.io/gorm"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var (
	FirebaseAuthClient        *auth.Client
	TradersFirebaseAuthClient *auth.Client
	UserIdToConsumerCache     map[string]GetConsumerResult
	UserIdToTraderCache       map[string]GetTraderResult
	TokenCache                *cache.Cache // Keeps token stored for 20 minutes
)

func SetupFirebase(accountKeyPath string) *auth.Client {
	serviceAccountKeyFilePath, err := filepath.Abs(accountKeyPath)
	if err != nil {
		log.Panicf("Unable to load serviceAccountKeys.json file: %v", err)
	}

	opt := option.WithCredentialsFile(serviceAccountKeyFilePath)
	//Firebase admin SDK initialization
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Panicf("Firebase Admin SDK load error: %v", err)
	}
	//Firebase Auth
	auth, err := app.Auth(context.Background())
	if err != nil {
		log.Panicf("Firebase Auth load error: %v", err)
	}
	TokenCache = cache.New(20*time.Minute, 20*time.Minute)

	return auth
}

// AuthMiddleware : to verify all authorized operations
func AuthMiddleware(ctx *gin.Context) {
	firebaseAuth := ctx.MustGet("firebaseAuth").(*auth.Client)
	authorizationToken := ctx.GetHeader("Authorization")
	idToken := strings.TrimSpace(strings.Replace(authorizationToken, "Bearer", "", 1))
	uid, done := getUid(ctx, idToken, firebaseAuth)
	if uid == "" || done {
		return
	}

	ctx.Set("FIREBASE_USER_UID", uid)

	var consumerCached, traderCached, isAdmin, isGuest bool
	var consumerId, traderId uint

	if cacheConsumer, ok := UserIdToConsumerCache[uid]; ok {
		if cacheConsumer.ID != 0 {
			consumerId = cacheConsumer.ID
			isGuest = cacheConsumer.IsGuest
		}
		consumerCached = ok
	}

	if cacheTrader, ok := UserIdToTraderCache[uid]; ok {
		if cacheTrader.ID != 0 {
			traderId = cacheTrader.ID
			isAdmin = cacheTrader.IsAdmin
		}
		traderCached = ok
	}

	var success bool
	var email string
	if !consumerCached || !traderCached {
		email, success = tryGetUserEmail(ctx, firebaseAuth, uid)
		if !success {
			return
		}
	}
	if !consumerCached {
		consumerId, isGuest, success = tryExtractConsumerIdFromUid(ctx, email, uid)
	}
	if !traderCached {
		traderId, isAdmin, success = tryExtractTraderIdFromUid(ctx, email, uid)
	}

	if consumerId == 0 && traderId == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error. User not found.")})
		ctx.Abort()
		return
	}

	if consumerId != 0 {
		ctx.Set("AUTHENTICATED_CONSUMER_ID", consumerId)
		ctx.Set("IS_GUEST", isGuest)
	}
	if traderId != 0 {
		ctx.Set("AUTHENTICATED_TRADER_ID", traderId)
		if isAdmin {
			ctx.Set("IS_ADMIN", isAdmin)
		}
	}
	ctx.Next()
}

// RequireAdminAuth : to verify only admins access specific endpoint
func RequireAdminAuth(ctx *gin.Context) {
	isAdmin, exists := ctx.Get("IS_ADMIN")

	if !exists || !isAdmin.(bool) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error. No sufficient permissions.")})
		ctx.Abort()
		return
	}
	ctx.Next()
}

func getUid(ctx *gin.Context, idToken string, firebaseAuth *auth.Client) (string, bool) {
	if idToken == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error - No id token found for this request (%v)", env.GetEnvVar("SERVICE_NAME"))})
		ctx.Abort()
		return "", true
	}
	if uid, found := TokenCache.Get(idToken); found {
		return uid.(string), false
	}
	//verify token
	token, err := firebaseAuth.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error - Token not verified, err: %v (%v)", err, env.GetEnvVar("SERVICE_NAME"))})
		ctx.Abort()
		return "", true
	}
	TokenCache.Set(idToken, token.UID, cache.DefaultExpiration)
	UserIdToConsumerCache = map[string]GetConsumerResult{}
	UserIdToTraderCache = map[string]GetTraderResult{}
	return token.UID, false
}

func tryExtractConsumerIdFromUid(ctx *gin.Context, email string, uid string) (id uint, isGuest bool, success bool) {
	var result GetConsumerResult
	err2 := ctx.MustGet("DB").(*gorm.DB).Raw("SELECT id, is_guest FROM consumers.consumers WHERE email = ?", email).Scan(&result).Error
	UserIdToConsumerCache[uid] = result
	if err2 != nil || result.ID == 0 {
		return 0, true, false
	}
	return result.ID, result.IsGuest, true
}

type GetTraderResult struct {
	ID      uint
	IsAdmin bool
}

type GetConsumerResult struct {
	ID      uint
	IsGuest bool
}

func tryExtractTraderIdFromUid(ctx *gin.Context, email string, uid string) (uint, bool, bool) {
	var result GetTraderResult
	err2 := ctx.MustGet("DB").(*gorm.DB).Raw("SELECT id, is_admin FROM traders.traders WHERE email = ?", email).Scan(&result).Error
	UserIdToTraderCache[uid] = result

	if err2 != nil || result.ID == 0 {
		return 0, false, false
	}
	return result.ID, result.IsAdmin, true
}

func tryGetUserEmail(ctx *gin.Context, firebaseAuth *auth.Client, uid string) (string, bool) {
	userRecord, err := firebaseAuth.GetUser(context.Background(), uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Authentication Error - User record not found: %v, (%v)", err, env.GetEnvVar("SERVICE_NAME"))})
		ctx.Abort()
		return "", false
	}
	ctx.Set("FIREBASE_USER_EMAIL", userRecord.Email)
	return userRecord.Email, true
}
