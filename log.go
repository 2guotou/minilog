package minilog

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"time"
)

type Logger struct {
	Dir    string       //日志存储目录
	File   string       //日志基础名称
	Date   string       //日志当前日期
	Writer *os.File     //日志文件的句柄
	Buffer chan logtext //日志的缓冲通道
	wlnfl  []string     //Which Level Need Filename & LineNumber
}

type logtext struct {
	date  string //日志日期
	level string //级别
	time  string //日志时间
	text  string //日志内容
}

const (
	logFlag = os.O_CREATE | os.O_WRONLY | os.O_APPEND //日志 创建 只写 追加
	logMode = 0666                                    //日志 可读 可写
	logDate = "2006-01-02"                            //日期按日切割的FORMAT
	logTime = "15:04:05"                              //时间格式3
)

//创建一个新的日志记录器
func NewLogger(dir string, filename string, bufferSize int64) (l Logger) {
	l.Date = time.Now().Format(logDate)
	l.File = filename
	l.Dir = path.Dir(dir + "/") //确保路径格式正确，避免路径分隔符或多或少
	l.Buffer = make(chan logtext, bufferSize)
	if writer, err := l.getWriter(); err != nil {
		panic("日志创建失败：" + err.Error())
	} else {
		l.Writer = writer
	}
	go l.flush()
	return
}

func (l *Logger) SetWlnfl(sets ...string) {
	l.wlnfl = sets
}

func (l *Logger) getWriter() (writer *os.File, err error) {
	return os.OpenFile(l.Dir+"/"+l.File+"."+l.Date+".log", logFlag, logMode)
}

// 将文本内容刷入日志buffer中
// 写满前为异步操作
// 写满后则同步堵塞
func (l *Logger) Write(level, text string) {
	if len(l.wlnfl) > 0 {
		if ok, _ := inStringArray(l.wlnfl, level); ok {
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
	}
}

func (l *Logger) Info(text string) {
	l.Write("INFO", text)
}

func (l *Logger) Error(text string) {
	l.Write("ERRO", text)
}

func (l *Logger) Debug(text string) {
	l.Write("DEBG", text)
}

func (l *Logger) Fatal(text string) {
	l.Write("FATL", text)
}

func (l *Logger) Access(text string) {
	l.Write("ACES", text)
}

// 将buffer里的内容逐次刷入磁盘
func (l *Logger) flush() {
	//日志刷入操作
	for {
		lt := <-l.Buffer
		//每次写入前判断一下日期是否一致，不一致则创建新日志文件
		if lt.date != l.Date {
			l.Date = lt.date
			if writer, err := l.getWriter(); err != nil {
				//使用系统 panic log，暴露出本次异常，最好配置报警；
				//errLog.Write("创建次日日志失败: " + m.file + " " + datestr) //发送错误报警
			} else {
				l.Writer.Close()
				l.Writer = writer
			}
		}
		fmt.Fprintf(l.Writer, "%s %s [%s] %s\n", lt.date, lt.time, lt.level, lt.text)
	}
}

func inStringArray(arr []string, dst string) (bool, int) {
	if len(arr) == 0 {
		return false, -1
	}
	for i, v := range arr {
		if v == dst {
			return true, i
		}
	}
	return false, -1
}
