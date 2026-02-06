package context

import (
	"context"
	"encoding/json"
	"errors"
)

// GetUserIdFromCtx 从 context 获取 userId
func GetUserIdFromCtx(ctx context.Context) (int64, error) {
	value := ctx.Value("userId")
	if value == nil {
		return 0, errors.New("userId not found in context")
	}

	switch v := value.(type) {
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, err
		}
		return i, nil
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, errors.New("invalid userId type")
	}
}
