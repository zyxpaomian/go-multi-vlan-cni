package log

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
	"runtime"
	"strings"
	"util/config"
)

// 封装logrus.Fields
type Fields logrus.Fields

func InitLog() {
	//格式化输出
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	//设备日志级别
	switch strings.ToUpper(config.GlobalConf.GetStr("server", "loglevel")) {
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.DebugLevel)
	}

	//设置输出路径
	logdir := config.GlobalConf.GetStr("server", "logdir")
	logfile := config.GlobalConf.GetStr("server", "logname")
	file, err := os.OpenFile(logdir+"/"+logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic("打开日志文件失败")
	}
	logrus.SetOutput(file)
}

func Debugln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Debugln(args...)
}

func Debugf(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Debugf(format, args...)
}

func Warnln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Warnln(args...)
}

func Warnf(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Warnf(format, args...)
}

func Infoln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Infoln(args...)
}

func Infof(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Infof(format, args...)
}

func Errorln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Errorln(args...)
}

func Errorf(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Errorf(format, args...)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	fields["gofile"] = fmt.Sprintf("[%s:%d] ", file, line)
	return logrus.WithFields(fields)
}
