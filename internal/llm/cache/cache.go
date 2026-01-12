package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl int) error
	Delete(ctx context.Context, key string) error
}

// generate hash key from request
func GenerateCacheKey(messages []interface{}, model string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"messages": messages,
		"model":    model,
	})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
