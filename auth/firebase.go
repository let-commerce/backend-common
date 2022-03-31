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
	UserIdToConsumerIdCache   map[string]uint
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

	var consumerCached, traderCached, isAdmin bool
	var consumerId, traderId uint

	if cacheConsumerId, ok := UserIdToConsumerIdCache[uid]; ok {
		log.Infof("found consumer id in cache : %v", cacheConsumerId)
		if cacheConsumerId != 0 {
			consumerId = cacheConsumerId
		}
		consumerCached = ok
	}

	if cacheTraderId, ok := UserIdToTraderCache[uid]; ok {
		log.Infof("found trader id in cache : %v", cacheTraderId)
		if cacheTraderId.ID != 0 {
			traderId = cacheTraderId.ID
			isAdmin = cacheTraderId.IsAdmin
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
		consumerId, success = tryExtractConsumerIdFromUid(ctx, email, uid)
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
	}
	if traderId != 0 {
		ctx.Set("AUTHENTICATED_TRADER_ID", traderId)
		if isAdmin {
			ctx.Set("IS_ADMIN", isAdmin)
		}
	}
	log.Infof("authenicated_consumer_id = %v, authenticated_trader_id = %v, is_admin = %v", consumerId, traderId, isAdmin)
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
	UserIdToConsumerIdCache = map[string]uint{}
	UserIdToTraderCache = map[string]GetTraderResult{}
	return token.UID, false
}

func tryExtractConsumerIdFromUid(ctx *gin.Context, email string, uid string) (uint, bool) {
	var result uint
	err2 := ctx.MustGet("DB").(*gorm.DB).Raw("SELECT id FROM consumers.consumers WHERE email = ?", email).Scan(&result).Error
	UserIdToConsumerIdCache[uid] = result
	if err2 != nil || result == 0 {
		return 0, false
	}
	return result, true
}

type GetTraderResult struct {
	ID      uint
	IsAdmin bool
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
