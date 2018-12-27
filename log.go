package minilog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	//LevelInfo 常规信息
	LevelInfo = "INFO"
	//LevelError 错误信息
	LevelError = "ERRO"
	//LevelDebug 调试信息
	LevelDebug = "DEBG"
	//LevelFatal 致命信息
	LevelFatal = "FATL"
	//LevelAccess 访问信息
	LevelAccess = "ACES"
)

//Logger ...
type Logger struct {
	Dir      string       //日志存储目录
	File     string       //日志基础名称
	Date     string       //日志当前日期
	Writer   *os.File     //日志文件的句柄
	Buffer   chan logtext //日志的缓冲通道
	Levels   map[string]*Level
	Callback callback
	close    chan bool //关闭指令
}

//Level 日志层级
type Level struct {
	WithFileLine bool     //是否带有文件名+行号
	Individual   bool     //是否独立
	Duplicate    bool     //在独立的基础上, 是保留母体日志的冗余
	Writer       *os.File //独立的文件句柄, 只有 individual==true 才会用
	Date         string   //当前如理日志的日期字符串, 用于比对创建新日志
}

type logtext struct {
	date  string   //日志日期
	level string   //级别
	time  string   //日志时间
	text  string   //日志内容
	call  callback //回调
	dry   bool     //是否是干写
}

type callback func(string, string)

const (
	logFlag = os.O_CREATE | os.O_WRONLY | os.O_APPEND //日志 创建 只写 追加
	logMode = 0666                                    //日志 可读 可写
	logDate = "2006-01-02"                            //日期按日切割的FORMAT
	logTime = "15:04:05"                              //时间格式3
)

//EmptyIns ...
var EmptyIns = []interface{}{}
var osSeparator = "/"

func init() {
	if runtime.GOOS == "windows" {
		osSeparator = "\\"
	}
}

//NewLogger 创建一个新的日志记录器
func NewLogger(dir string, filename string, bufferSize int64) *Logger {
	l := new(Logger)
	l.Date = time.Now().Format(logDate)
	l.File = filename
	l.Levels = map[string]*Level{}
	l.Dir = strings.TrimRight(dir, osSeparator) + osSeparator //确保路径格式正确，避免路径分隔符或多或少
	l.Buffer = make(chan logtext, bufferSize)
	l.close = make(chan bool)
	if writer, err := l.getWriter(l.Date, ""); err != nil {
		panic("日志创建失败：" + err.Error())
	} else {
		l.Writer = writer
	}
	go l.flush()
	return l
}

//LevelsSet 批量层级配置
func (l *Logger) LevelsSet(ls map[string]*Level) {
	l.Levels = ls
}

//LevelSet 设置某个层级的配置
//withFileLine 日志中追加 文件名和行号
//individual   保有独立的日志文件
//duplicate    保有母日志文件中的冗余
func (l *Logger) LevelSet(level string, withFileLine, individual, duplicate bool) {
	l.Levels[level] = &Level{
		WithFileLine: withFileLine,
		Individual:   individual,
		Duplicate:    duplicate,
	}
}

//WithFileLine 有哪些 Level 需要记录文件和行号
//不再鼓励使用, 请直接使用 LevelSet 完成配置
func (l *Logger) WithFileLine(sets ...string) {
	for _, level := range sets {
		if levelSetting, exist := l.Levels[level]; exist {
			levelSetting.WithFileLine = true
		} else {
			l.Levels[level] = &Level{
				WithFileLine: true,
			}
		}
	}
}

func (l *Logger) getWriter(date, level string) (writer *os.File, err error) {
	if level != "" {
		level = "." + level
	}
	logfile := fmt.Sprintf("%s%s.%s%s.log", l.Dir, l.File, date, level)
	fmt.Println("minilog: create log, filename=" + logfile)
	return os.OpenFile(logfile, logFlag, logMode)
}

// 将文本内容刷入日志buffer中
// 写满前为异步操作
// 写满后则同步堵塞
func (l *Logger) Write(level, text string, ins []interface{}, c callback) {
	if len(ins) > 0 {
		text = fmt.Sprintf(text, ins...)
	}
	if levelSetting, exist := l.Levels[level]; exist {
		if levelSetting.WithFileLine {
			_, f, l, _ := runtime.Caller(2)
			text = fmt.Sprintf("%s [%s:%d]", text, f, l)
		}
	}

	t := time.Now()
	l.Buffer <- logtext{
		date:  t.Format(logDate),
		time:  t.Format(logTime),
		level: level,
		text:  text,
		call:  c,
	}
}

//DryWrite 直接写一行字符串到日志
func (l *Logger) DryWrite(text string) {
	t := time.Now()
	l.Buffer <- logtext{
		date: t.Format(logDate),
		time: t.Format(logTime),
		text: text,
		dry:  true,
	}
}

//Info 信息类消息
func (l *Logger) Info(text string, ins ...interface{}) {
	l.Write(LevelInfo, text, ins, l.Callback)
}

//Error 错误类消息
func (l *Logger) Error(text string, ins ...interface{}) {
	l.Write(LevelError, text, ins, l.Callback)
}

//Debug 调试类消息
func (l *Logger) Debug(text string, ins ...interface{}) {
	l.Write(LevelDebug, text, ins, l.Callback)
}

//Fatal 致命错误
func (l *Logger) Fatal(text string, ins ...interface{}) {
	l.Write(LevelFatal, text, ins, l.Callback)
}

//Access ...
func (l *Logger) Access(text string, ins ...interface{}) {
	l.Write(LevelAccess, text, ins, l.Callback)
}

//AccessCall ...
func (l *Logger) AccessCall(text string, c callback) {
	l.Write(LevelAccess, text, EmptyIns, c)
}

//获取 buffer 中的数据, 并根据 close channel 决定是否退出
func (l *Logger) flush() {

	defer func() {
		l.Writer.Close()
		for _, level := range l.Levels {
			level.Writer.Close()
		}
		l.close <- true
	}()

	for {
		select {
		case <-l.close:
			return
		case lt := <-l.Buffer:
			l.flushing(lt)
		}
	}
}

// 将buffer里的内容逐次刷入磁盘
func (l *Logger) flushing(lt logtext) {
	//=======================[层级日志操作]==========================
	levelSetting, exist := l.Levels[lt.level]
	if exist && levelSetting.Individual {

		if lt.date != levelSetting.Date || levelSetting.Writer == nil {
			levelSetting.Date = lt.date
			if writer, err := l.getWriter(levelSetting.Date, lt.level); err == nil {
				levelSetting.Writer.Close()
				levelSetting.Writer = writer
			} else {
				fmt.Println("minilog: create log file faild, err=" + err.Error())
			}
		}
		if levelSetting.Writer != nil {
			fmt.Fprintf(levelSetting.Writer, "%s %s [%s] %s\n", lt.date, lt.time, lt.level, lt.text)
		}

		if !levelSetting.Duplicate {
			//Duplication 依赖于 Individual==true !!!!
			//如果有后续调用则调用该函数
			if lt.call != nil {
				lt.call(lt.text, lt.date+" "+lt.time)
			}
			//如果有后续调用则调用该函数
			if lt.call != nil {
				lt.call(lt.text, lt.date+" "+lt.time)
			}
			return //不需要在母体日志中冗余, 则跳过后续环节
		}
	}

	//=======================[母体日志操作]===========================
	//每次写入前判断一下日期是否一致，不一致则创建新日志文件
	if lt.date != l.Date {
		l.Date = lt.date
		if writer, err := l.getWriter(l.Date, ""); err == nil {
			l.Writer.Close()
			l.Writer = writer
		} else {
			fmt.Println("minilog: create log file faild, err=" + err.Error())
		}
	}
	if l.Writer != nil {
		if lt.dry {
			fmt.Fprintf(l.Writer, "%s\n", lt.text)
		} else {
			fmt.Fprintf(l.Writer, "%s %s [%s] %s\n", lt.date, lt.time, lt.level, lt.text)
		}
	}

	//如果有后续调用则调用该函数
	if lt.call != nil {
		lt.call(lt.text, lt.date+" "+lt.time)
	}
}

// Close 关闭日志, 尽可能将消息落地
// maxWait 最多等待的毫秒数(不严格的接近1毫秒)
func (l *Logger) Close(maxWait int) {
	for i := 0; i < maxWait && len(l.Buffer) > 0; i++ {
		time.Sleep(time.Millisecond)
	}
	//发送结束信息, 随即堵塞等待真实关闭的信号
	l.close <- true
	<-l.close
}
