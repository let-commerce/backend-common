package env

import (
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
)

func InitEnvFile() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Panicf("Can't load .env: %v", err)
	}
	env := GetEnvVar("ENV")
	if env == "local" || env == "" {
		log.Info("loading local.env file")
		err := godotenv.Load("local.env")
		if err != nil {
			log.Panicf("Can't load local.env: %v", err)
		}
	}
}

func GetEnvVar(key string) string {
	return os.Getenv(key)
}

func MustGetEnvVar(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Panicf("Can't find evnvironment variable %v", key)
	}
	return value
}
