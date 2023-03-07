package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	syslog "log"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/jtolds/gls"
)

// Level 日志级别
type Level int

const (
	//FATAL 日志等级0
	FATAL Level = iota
	//ERROR 日志等级1
	ERROR
	//WARNING 日志等级2
	WARNING
	//INFO 日志等级3
	INFO
	//DEBUG 日志等级4
	DEBUG
	//ALL 日志等级5
	ALL
)

// Log 扩展系统logger
// 实现trace_id跟踪
// 实现日志切割/删除
type Log struct {
	logger         *syslog.Logger
	gls            *gls.ContextManager // Goroutine local storage
	config         Config
	rotateTime     string      // .log -> .log.(rotateTime)
	ch             chan string // log事件
	chToKafka2Algo chan string // 搜索推荐落入数仓的数据
	mu             sync.Mutex
}

// Config 日志配置
type Config struct {
	Type          string `json:"type"`
	Level         string `json:"level"`
	Dir           string `json:"dir"`
	FileName      string `json:"filename"`
	RotateByDaily bool   `json:"rotateByDaily"`
	RotateByHour  bool   `json:"rotateByHour"`
	KeepDays      int    `json:"keepDays"`
}

// 单例
var l Log

func init() {
	// 设置默认输出logger
	conf := Config{
		Type: "std",
	}
	SetLogger(newLogger(conf))
}

// Init Init
func Init(path string) error {
	var conf Config
	res, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New("error opening conf file=" + path)
	}
	if err := json.Unmarshal(res, &conf); err != nil {
		msg := fmt.Sprintf("error parsing conf file=%s, err=%s", path, err.Error())
		return errors.New(msg)
	}
	return InitByConf(conf)
}

// InitByConf InitByConf
func InitByConf(conf Config) error {
	SetConfig(conf)
	SetLogger(newLogger(conf))
	SetRotateTime(getCurrentTime(conf))
	SetGls(gls.NewContextManager())

	// 日志切割
	if conf.Type == "file" {
		go rotateDaemon()
	}

	return nil
}

// Go 代替go关键字，协程中的日志自动记住traceId
func Go(cb func()) {
	gls.Go(cb)
}

// Wrap Goroutine local storage 设置trace_id后执行函数
func Wrap(traceID string, cb func()) {
	l.gls.SetValues(gls.Values{"TraceID": traceID}, cb)
}

// SetConfig SetConfig
func SetConfig(conf Config) {
	l.config = conf
}

// SetLogger SetLogger
func SetLogger(logger *syslog.Logger) {
	l.logger = logger
}

// SetRotateTime SetRotateTime
func SetRotateTime(rotateTime string) {
	l.rotateTime = rotateTime
}

// SetCh SetCh
func SetCh(ch chan string) {
	l.ch = ch
}

// SetGls SetGls
func SetGls(gls *gls.ContextManager) {
	l.gls = gls
}

// SetChKafka2Algo SetChKafka2Algo
func SetChKafka2Algo(ch chan string) {
	l.chToKafka2Algo = ch
}

//GetTraceID http：从goroutine local storage 取
/**
* http：从goroutine local storage 取
* main：从当前进程取
 */
func GetTraceID() string {
	var traceID string
	if v, ok := l.gls.GetValue("TraceID"); ok {
		traceID = v.(string)
	} else {
		traceID = strconv.Itoa(os.Getpid())
	}

	return traceID
}

// GetLevel GetLevel
func GetLevel() string {
	return l.config.Level
}

// Output 以下日志输出函数
// 规定所有非格式化输出参数为map，方便合并trace_id以及
// output必须制定level
func Output(args map[string]interface{}) {
	if val, ok := args["level"].(string); ok {
		level := stringToLevel(val)
		if level > stringToLevel(l.config.Level) {
			return
		}
	}
	output(args)
}

// Debug Debug
func Debug(args map[string]interface{}) {
	print(DEBUG, args)
}

// DebugThrift DebugThrift
func DebugThrift(args interface{}) {
	printThrift(DEBUG, args)
}

// Debugf Debugf
func Debugf(format string, args ...interface{}) {
	printf(DEBUG, format, args...)
}

// Info Info
func Info(args map[string]interface{}) {
	print(INFO, args)
}

// Infof Infof
func Infof(format string, args ...interface{}) {
	printf(INFO, format, args...)
}

// Warning Warning
func Warning(args map[string]interface{}) {
	print(WARNING, args)
}

// Warningf Warningf
func Warningf(format string, args ...interface{}) {
	printf(WARNING, format, args...)
}

// Error Error
func Error(args map[string]interface{}) {
	print(ERROR, args)
}

// Errorf Errorf
func Errorf(format string, args ...interface{}) {
	printf(ERROR, format, args...)
}

// Fatal Fatal
func Fatal(args map[string]interface{}) {
	print(FATAL, args)
	os.Exit(1)
}

// Fatalf Fatalf
func Fatalf(format string, args ...interface{}) {
	printf(FATAL, format, args...)
	os.Exit(1)
}

/*
* 非格式化输出，合并trace_id
 */

func print(level Level, m map[string]interface{}) {
	if level > stringToLevel(l.config.Level) {
		return
	}

	m["level"] = levelToString(level)
	output(m)
}

func printThrift(level Level, msg interface{}) {
	if level > stringToLevel(l.config.Level) {
		return
	}
	m := make(map[string]interface{})
	byteMsg, err := json.Marshal(msg)
	if err != nil {
		m["err"] = err
	} else {
		m["msg"] = string(byteMsg)
	}
	m["level"] = levelToString(level)
	output(m)
}

func printf(level Level, format string, args ...interface{}) {
	if level > stringToLevel(l.config.Level) {
		return
	}

	m := map[string]interface{}{}
	m["msg"] = fmt.Sprintf(format, args...)
	m["level"] = levelToString(level)

	output(m)
}

func output(m map[string]interface{}) {
	m["trace_id"] = GetTraceID()
	// mysql当前文件直接传进来，比较准确
	if _, ok := m["file"]; !ok {
		m["file"] = getCurFileName()
	}
	m["timestamp"] = time.Now().Format("2006-01-02 15:04:05.999")
	m["host"], _ = os.Hostname()

	if l.config.Type != "none" {
		l.logger.Println(mapToStr(m))
	}
	// 满了防止阻塞
	//if l.ch != nil && len(l.ch) < 10000 {
	//	l.ch <- golib.ToString(m)
	//} else {
	//	fmt.Println("log ch 异常")
	//}
}

// OutPutToKafka2Algo OutPutToKafka2Algo
func OutPutToKafka2Algo(m map[string]interface{}) {
	m["timestamp"] = time.Now().Format("2006-01-02 15:04:05.999")
	m["host"], _ = os.Hostname()
	if l.chToKafka2Algo != nil && len(l.chToKafka2Algo) < 10000 {
		l.chToKafka2Algo <- ToString(m)
	} else {
		fmt.Println("OutPutToKafka2Algo 异常")
	}
}

func newLogger(conf Config) *syslog.Logger {
	flag := syslog.Ldate | syslog.Ltime | syslog.Lmicroseconds
	var output io.Writer
	if conf.Type == "file" {
		path := path.Join(conf.Dir, conf.FileName)
		fd, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			syslog.Fatal(err)
			os.Exit(1)
		}
		output = fd
	} else {
		output = os.Stdout
	}

	logger := syslog.New(output, "", flag)

	return logger
}

func rotateDaemon() {
	for {
		time.Sleep(time.Second * 1)
		// 切割
		currentTime := getCurrentTime(l.config)
		if currentTime != l.rotateTime {
			old := path.Join(l.config.Dir, l.config.FileName)
			new := path.Join(l.config.Dir, l.config.FileName+fmt.Sprintf(".%s", l.rotateTime))
			err := os.Rename(old, new)
			if err != nil {
				Warningf("rotateDaemon failed : %s", err.Error())
			}
			// 重新设置logger与rotateTime
			SetLogger(newLogger(l.config))
			SetRotateTime(currentTime)
		}

		// 删除
		files, err := ioutil.ReadDir(l.config.Dir)
		// 保留n天
		minKeepTime := time.Now().AddDate(0, 0, -l.config.KeepDays)
		reg := regexp.MustCompile("\\.log\\.20[0-9]{8}")
		if err == nil {
			for _, file := range files {
				if reg.FindString(file.Name()) != "" && shouldDel(file.Name(), minKeepTime) {
					os.Remove(path.Join(l.config.Dir, file.Name()))
				}
			}
		}
	}
}
