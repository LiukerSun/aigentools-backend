package services

import (
	"aigentools-backend/internal/database"
	"time"
)

const denylistPrefix = "denylist:"

func AddToDenylist(tokenString string, expiration time.Duration) error {
	key := denylistPrefix + tokenString
	return database.RedisClient.Set(database.Ctx, key, 1, expiration).Err()
}

func IsDenylisted(tokenString string) (bool, error) {
	key := denylistPrefix + tokenString
	val, err := database.RedisClient.Get(database.Ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" { // key does not exist
			return false, nil
		}
		return false, err
	}
	return val != "", nil
}
