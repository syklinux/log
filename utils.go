package log

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ShortStr 缩短文件名，最多显示3级
func ShortStr(str string, separator string, n int) string {
	strs := strings.Split(str, separator)
	// 缩短文件名，最多显示3级
	if n > len(strs) {
		n = len(strs)
	}
	result := ""
	for i := n; i > 0; i-- {
		result += strs[len(strs)-i] + separator
	}
	return strings.TrimSuffix(result, separator)
}

// FormatDurationToMs 时间间隔转为毫秒
func FormatDurationToMs(d time.Duration) string {
	return fmt.Sprintf("%.2f", float64(d.Nanoseconds())/float64(time.Millisecond))
}

// ToString interface转string
func ToString(value interface{}) (s string) {
	if reflect.TypeOf(value) == nil {
		return ""
	}

	switch v := value.(type) {
	case bool:
		s = strconv.FormatBool(v)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		s = strconv.FormatInt(int64(v), 10)
	case int8:
		s = strconv.FormatInt(int64(v), 10)
	case int16:
		s = strconv.FormatInt(int64(v), 10)
	case int32:
		s = strconv.FormatInt(int64(v), 10)
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint:
		s = strconv.FormatUint(uint64(v), 10)
	case uint8:
		s = strconv.FormatUint(uint64(v), 10)
	case uint16:
		s = strconv.FormatUint(uint64(v), 10)
	case uint32:
		s = strconv.FormatUint(uint64(v), 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case string:
		s = v
	case []byte:
		s = string(v)
	case error:
		s = v.Error()
	default:
		b, _ := json.Marshal(v)
		s = string(b)
	}
	return s
}
