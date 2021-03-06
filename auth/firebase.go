package auth

import (
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/let-commerce/backend-common/env"
	"github.com/orcaman/concurrent-map"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"gorm.io/gorm"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	FirebaseAuthClient          *auth.Client
	BackofficeAuthClient        *auth.Client
	UserIdToConsumerCache       cmap.ConcurrentMap //using concurrent map to prevent thread safe issues -  map[string]GetConsumerResult
	UserIdToBackofficeUserCache cmap.ConcurrentMap //using concurrent map to prevent thread safe issues - map[string]GetBackofficeUserResult
)

func Init() {
	UserIdToConsumerCache = cmap.New()       //using concurrent map to prevent thread safe issues
	UserIdToBackofficeUserCache = cmap.New() //using concurrent map to prevent thread safe issues
}

func SetupAllFirebase(accountKeyPath string, backofficeKeyPath string) (consumersFirebase *auth.Client, backofficeFirebase *auth.Client) {
	consumersFirebase = SetupFirebase(accountKeyPath)
	backofficeFirebase = SetupFirebase(backofficeKeyPath)
	return consumersFirebase, backofficeFirebase
}

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

	return auth
}

// AuthMiddleware : to verify all authorized operations
func AuthMiddleware(ctx *gin.Context) {
	firebaseAuth := ctx.MustGet("firebaseAuth").(*auth.Client)
	backofficeFirebaseAuth := ctx.MustGet("backofficeFirebaseAuth").(*auth.Client)
	authorizationToken := ctx.GetHeader("Authorization")
	requestContext := ctx.GetHeader("RequestContext")
	jwtToken := strings.TrimSpace(strings.Replace(authorizationToken, "Bearer", "", 1))

	var uid string
	var done bool

	if requestContext == "Backoffice" {
		uid, done = getUid(ctx, jwtToken, backofficeFirebaseAuth)
	} else {
		uid, done = getUid(ctx, jwtToken, firebaseAuth)
	}
	if uid == "" || done {
		return
	}

	ctx.Set("FIREBASE_USER_UID", uid)
	ctx.Next()
}

// RequireAuth : to verify all authorized operations, there exist a consumer id
func RequireAuth(ctx *gin.Context) {
	uidValue, exists := ctx.Get("FIREBASE_USER_UID")
	if !exists {
		return
	}
	uid := uidValue.(string)
	firebaseAuth := ctx.MustGet("firebaseAuth").(*auth.Client)
	backofficeFirebaseAuth := ctx.MustGet("backofficeFirebaseAuth").(*auth.Client)
	requestContext := ctx.GetHeader("RequestContext")

	var consumerCached, backofficeCached, isAdmin, isGuest, success bool
	isGuest = true
	var consumerId, backofficeUserId uint
	var email string

	if requestContext != "Backoffice" {
		if cacheConsumerValue, ok := UserIdToConsumerCache.Get(uid); ok {
			cacheConsumer := cacheConsumerValue.(GetConsumerResult)
			if cacheConsumer.ID != 0 {
				consumerId = cacheConsumer.ID
				isGuest = cacheConsumer.IsGuest
			}
			consumerCached = ok
		}
		if !consumerCached {
			email, success = tryGetUserEmail(ctx, firebaseAuth, uid)
			if !success {
				return
			}
			consumerId, isGuest, success = tryExtractConsumerIdFromUid(ctx, email, uid)
		}
	} else {
		if cacheBackofficeUserValue, ok := UserIdToBackofficeUserCache.Get(uid); ok {
			cacheBackofficeUser := cacheBackofficeUserValue.(GetBackofficeUserResult)
			if cacheBackofficeUser.ID != 0 {
				backofficeUserId = cacheBackofficeUser.ID
				isAdmin = cacheBackofficeUser.IsAdmin
			}
			backofficeCached = ok
		}
		if !backofficeCached {
			email, success = tryGetUserEmail(ctx, backofficeFirebaseAuth, uid)
			if !success {
				return
			}
			backofficeUserId, isAdmin, success = tryExtractBackofficeUserIdFromUid(ctx, email, uid)
		}
	}

	if consumerId == 0 && backofficeUserId == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error. User not found.")})
		ctx.Abort()
		return
	}

	if consumerId != 0 {
		ctx.Set("AUTHENTICATED_CONSUMER_ID", consumerId)
		ctx.Set("IS_GUEST", isGuest)
	}
	if backofficeUserId != 0 {
		ctx.Set("AUTHENTICATED_BACKOFFICE_USER_ID", backofficeUserId)
		ctx.Set("IS_ADMIN", isAdmin)
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

func getUid(ctx *gin.Context, jwtToken string, firebaseAuth *auth.Client) (string, bool) {
	if jwtToken == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error - No id token found for this request (%v)", env.GetEnvVar("SERVICE_NAME"))})
		ctx.Abort()
		return "", true
	}
	//verify token
	token, err := firebaseAuth.VerifyIDToken(ctx, jwtToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error - Token not verified, err: %v (%v)", err, env.GetEnvVar("SERVICE_NAME"))})
		ctx.Abort()
		return "", true
	}
	return token.UID, false
}

func tryExtractConsumerIdFromUid(ctx *gin.Context, email string, uid string) (id uint, isGuest bool, success bool) {
	var result GetConsumerResult
	err2 := ctx.MustGet("DB").(*gorm.DB).Raw("SELECT id, is_guest FROM consumers.consumers WHERE email = ?", email).Scan(&result).Error
	UserIdToConsumerCache.Set(uid, result)
	if err2 != nil || result.ID == 0 {
		return 0, true, false
	}
	return result.ID, result.IsGuest, true
}

type GetBackofficeUserResult struct {
	ID      uint
	IsAdmin bool
}

type GetConsumerResult struct {
	ID      uint
	IsGuest bool
}

func tryExtractBackofficeUserIdFromUid(ctx *gin.Context, email string, uid string) (uint, bool, bool) {
	var result GetBackofficeUserResult
	err2 := ctx.MustGet("DB").(*gorm.DB).Raw("SELECT id, is_admin FROM consumers.back_office_users WHERE email = ?", email).Scan(&result).Error
	UserIdToBackofficeUserCache.Set(uid, result)

	if err2 != nil || result.ID == 0 {
		return 0, false, false
	}
	return result.ID, result.IsAdmin, true
}

func tryGetUserEmail(ctx *gin.Context, firebaseAuth *auth.Client, uid string) (string, bool) {
	userRecord, err := firebaseAuth.GetUser(ctx, uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Authentication Error - User record not found: %v, (%v)", err, env.GetEnvVar("SERVICE_NAME"))})
		ctx.Abort()
		return "", false
	}
	ctx.Set("FIREBASE_USER_EMAIL", userRecord.Email)
	return userRecord.Email, true
}
