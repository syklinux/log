package log

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// FormatTimeDay 日期格式
const FormatTimeDay string = "20060102"

// FormatTimeHour 时间格式
const FormatTimeHour string = "2006010215"

func getDayTime(t time.Time) string {
	return t.Format(FormatTimeDay)
}

func getHourTime(t time.Time) string {
	return t.Format(FormatTimeHour)
}

func getCurrentTime(conf Config) string {
	var rotateTime string
	if conf.RotateByDaily == true {
		rotateTime = getDayTime(time.Now())
	} else if conf.RotateByHour == true {
		rotateTime = getHourTime(time.Now())
	}
	return rotateTime
}

func levelToString(level Level) string {
	switch level {
	case FATAL:
		return "FATAL"
	case ERROR:
		return "ERROR"
	case WARNING:
		return "WARNING"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	}
	return "ALL"
}

func stringToLevel(level string) Level {
	switch level {
	case "FATAL":
		return FATAL
	case "ERROR":
		return ERROR
	case "WARNING":
		return WARNING
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	}
	return ALL
}

func getTimeInt(now time.Time) uint64 {
	return uint64(now.Year())*1000000 + uint64(now.Month())*10000 + uint64(now.Day())*100 + uint64(now.Hour())
}

func shouldDel(fileName string, keepTime time.Time) bool {

	// project.log.2019071016 -> 2019071016
	strs := strings.Split(fileName, ".")
	tint, err := strconv.Atoi(strs[len(strs)-1])
	if err != nil {
		return false
	}

	if uint64(tint) < getTimeInt(keepTime) {
		return true
	}

	return false
}

func getbuf() *bytes.Buffer {
	return &bytes.Buffer{}
}

func getCurFileName() string {

	pc, file, line, _ := runtime.Caller(4)
	function := runtime.FuncForPC(pc)

	// 缩短文件名，最多显示3级
	fileName := ShortStr(file, "/", 3)

	return fileName + ":" + strconv.Itoa(line) + "::" + function.Name()
}

func contentToBuffer(header string, body string) *bytes.Buffer {

	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, header)
	fmt.Fprintf(buf, " ")
	fmt.Fprintf(buf, body)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}

	return buf
}

func mapToStr(m map[string]interface{}) string {
	var str string

	keys := []string{"level", "file", "trace_id", "msg", "cost", "timestamp", "host", "data", "client_ip", "type"}

	// 按顺序展示key
	for _, k := range keys {
		if v, ok := m[k]; ok {
			str = str + fmt.Sprintf("%v=%s", k, ToString(v))
			str = str + "||"
		}
	}

	return strings.TrimSuffix(str, "||")
}
