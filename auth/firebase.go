package auth

import (
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"fmt"
	"github.com/gin-gonic/gin"
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
	FirebaseAuthClient      *auth.Client
	UserIdToConsumerIdCache map[string]int
	TokenCache              *cache.Cache // Keeps token stored for 20 minutes
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
	consumerId, success := tryExtractConsumerIdFromUid(ctx, firebaseAuth, uid)
	if !success {
		return
	}
	ctx.Set("AUTHENTICATED_CONSUMER_ID", consumerId)
	ctx.Next()
}

func getUid(ctx *gin.Context, idToken string, firebaseAuth *auth.Client) (string, bool) {
	if idToken == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication Error - No id token found for this request"})
		ctx.Abort()
		return "", true
	}
	if uid, found := TokenCache.Get(idToken); found {
		return uid.(string), false
	}
	//verify token
	token, err := firebaseAuth.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Authentication Error - Token not verified, err: %v", err)})
		ctx.Abort()
		return "", true
	}
	TokenCache.Set(idToken, token.UID, cache.DefaultExpiration)
	UserIdToConsumerIdCache = map[string]int{}
	return token.UID, false
}

func tryExtractConsumerIdFromUid(ctx *gin.Context, firebaseAuth *auth.Client, uid string) (int, bool) {
	if consumerId, ok := UserIdToConsumerIdCache[uid]; ok {
		return consumerId, true
	}
	userRecord, err2 := firebaseAuth.GetUser(context.Background(), uid)
	if err2 != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Authentication Error - User record not found: %v", err2)})
		ctx.Abort()
		return 0, false
	}
	ctx.Set("FIREBASE_USER_EMAIL", userRecord.Email)

	var result int
	err2 = ctx.MustGet("DB").(*gorm.DB).Raw("SELECT id FROM consumers.consumers WHERE email = ?", userRecord.Email).Scan(&result).Error
	if err2 != nil || result == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Authentication Error - Consumer record not found for email: %v, err: %v", userRecord.Email, err2)})
		ctx.Abort()
		return 0, false
	}
	UserIdToConsumerIdCache[uid] = result
	return result, true
}
