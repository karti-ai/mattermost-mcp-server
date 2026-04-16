package params

import (
	"fmt"
	"strconv"
)

func GetString(args map[string]any, key string) (string, error) {
	val, ok := args[key].(string)
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	return val, nil
}

func GetOptionalString(args map[string]any, key, defaultVal string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultVal
}

func GetStringSlice(args map[string]any, key string) []string {
	val, ok := args[key]
	if !ok {
		return nil
	}
	sliceVal, ok := val.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(sliceVal))
	for _, item := range sliceVal {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func GetPagination(args map[string]any, defaultPageSize int64) (page, pageSize int) {
	return int(GetOptionalInt(args, "page", 1)), int(GetOptionalInt(args, "perPage", defaultPageSize))
}

func ToInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case float64:
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func GetIndex(args map[string]any, key string) (int64, error) {
	val, exists := args[key]
	if !exists {
		return 0, fmt.Errorf("%s is required", key)
	}

	if i, ok := ToInt64(val); ok {
		return i, nil
	}

	if s, ok := val.(string); ok {
		return 0, fmt.Errorf("%s must be a valid integer (got %q)", key, s)
	}

	return 0, fmt.Errorf("%s must be a number or numeric string", key)
}

func GetInt64Slice(args map[string]any, key string) ([]int64, error) {
	raw, ok := args[key].([]any)
	if !ok {
		return nil, fmt.Errorf("%s (array of IDs) is required", key)
	}
	out := make([]int64, 0, len(raw))
	for _, v := range raw {
		id, ok := ToInt64(v)
		if !ok {
			return nil, fmt.Errorf("invalid ID in %s array", key)
		}
		out = append(out, id)
	}
	return out, nil
}

func GetOptionalInt(args map[string]any, key string, defaultVal int64) int64 {
	val, exists := args[key]
	if !exists {
		return defaultVal
	}
	if i, ok := ToInt64(val); ok {
		return i
	}
	return defaultVal
}

func GetOptionalBool(args map[string]any, key string, defaultVal bool) bool {
	val, exists := args[key]
	if !exists {
		return defaultVal
	}

	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}

	return defaultVal
}
