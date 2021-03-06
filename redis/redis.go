package redis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/let-commerce/backend-common/env"
	log "github.com/sirupsen/logrus"
)

// for more operations: https://lzone.de/cheat-sheet/Redis

func RedisConnect() redis.Conn {
	conn, err := redis.Dial("tcp", env.MustGetEnvVar("REDIS_URL"))
	if err != nil {
		log.Panicf("Can't connect to redis client: %v", err)
	}
	return conn
}

func SetValue(conn redis.Conn, key string, value interface{}) {
	_, err := conn.Do("SET", key, value)
	if err != nil {
		log.Panicf("Got error while setting redis value: %v", err)
	}
}

func GetStringValue(conn redis.Conn, key string) string {
	if !Exists(conn, key) {
		return ""
	}
	value, err := redis.String(conn.Do("GET", key))
	if err != nil {
		log.Panicf("Got error while getting redis value: %v", err)
	}
	return value
}

func DeleteKey(conn redis.Conn, key string) {
	_, err := conn.Do("DEL", key)
	if err != nil {
		log.Panicf("Got error while deleting redis key: %v", err)
	}
}

func Exists(conn redis.Conn, key string) bool {
	value, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		log.Panicf("Got error while checking if key exists: %v", err)
	}
	return value
}

func SetTTL(conn redis.Conn, key string, secondsTTL float32) {
	_, err := conn.Do("EXPIRE", key, secondsTTL)
	if err != nil {
		log.Panicf("Got error while setting ttl: %v", err)
	}
}

func Increase(conn redis.Conn, key string) {
	_, err := conn.Do("INCR", key)
	if err != nil {
		panic(err)
	}
}

func Decrease(conn redis.Conn, key string) {
	_, err := conn.Do("DECR", key)
	if err != nil {
		panic(err)
	}
}

func PushToList(conn redis.Conn, key string, value interface{}) {
	_, err := conn.Do("LPUSH", key, value)
	if err != nil {
		panic(err)
	}
}

func GetListElement(conn redis.Conn, key string, index int) string {
	value, err := redis.String(conn.Do("LINDEX", key, index))
	if err != nil {
		panic(err)
	}
	return value
}

func SetHashMapValue(conn redis.Conn, hashName string, key string, value interface{}) {
	_, err := conn.Do("HSET", hashName, key, value)
	if err != nil {
		panic(err)
	}
}

func GetHashMapValue(conn redis.Conn, hashName string, key string) string {
	value, err := redis.String(conn.Do("HGET", hashName, key))
	if err != nil {
		panic(err)
	}
	return value
}
