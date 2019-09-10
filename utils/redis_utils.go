package utils

import (
	"github.com/go-redis/redis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"regexp"
	"time"
)

var redisClient *redis.Client
var replaceReg *regexp.Regexp

func InitRedis(redisAddr string) {
	opt, err := redis.ParseURL(redisAddr)
	if err != nil {
		panic(err)
	}
	redisClient = redis.NewClient(opt)
	replaceReg, err = regexp.Compile("[-+]+")
	if err != nil {
		log.Fatalf("phone regex creation error: %v", err)
	}
}
func SetKey(key, value string, duration time.Duration) error {
	key = replaceReg.ReplaceAllString(key, "")
	err := redisClient.Set(key, value, duration).Err()
	if err != nil {
		return status.Error(codes.Internal, "INTERNAL:REDIS_ERROR")
	}
	return nil
}
func GetKey(key string) (val string, err error) {
	key = replaceReg.ReplaceAllString(key, "")
	val, err = redisClient.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", status.Error(codes.InvalidArgument, "NON_EXISTS")
		}
		return "", status.Error(codes.Internal, "INTERNAL:REDIS_ERROR")
	}
	return val, nil
}

func CheckPhoneCode(codeType, phone, code string) error {
	key := codeType + phone
	val, err := GetKey(key)
	if err != nil {
		return err
	}
	if val != code {
		return status.Error(codes.InvalidArgument, "INVALID_CODE")
	}
	return nil
}