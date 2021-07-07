package logger

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

const (
	MODULE_MANAGER          = "[Docker MANAGER]"
	MODULE_SCHEDULER        = "[Docker Scheduler]"
	MODULE_USERCONTROLLER   = "[Docker User Controller]"
	MODULE_HANDLER_REGISTER = "[Docker Handler Register]"
	MODULE_UDS_SERVER       = "[Docker UDS Server]"
	MODULE_DOCKER_SERVER    = "[Docker Docker Server]"
	MODULE_SECURITY_ENV     = "[Docker Security Env]"
	MODULE_CONTRACT_MANAGER = "[Docker Contract Manager]"
)

func NewDockerLogger(name string) *zap.SugaredLogger {
	encoder := getEncoder()
	writeSyncer := getLogWriter()

	logLevel := zapcore.DebugLevel
	if config.LogLevel == "INFO" {
		logLevel = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		logLevel,
	)

	logger := zap.New(core).Named(name)
	defer logger.Sync()

	if config.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}

	sugarLogger := logger.Sugar()

	return sugarLogger
}

func getEncoder() zapcore.Encoder {

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    CustomLevelEncoder,
		EncodeTime:     CustomTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {

	hook := &lumberjack.Logger{
		Filename:   config.LogFile, //日志文件存放目录
		MaxSize:    100,            //文件大小限制,单位MB
		MaxBackups: 5,              //最大保留日志文件数量
		MaxAge:     30,             //日志文件保留天数
		Compress:   false,          //是否压缩处理
	}

	var syncer zapcore.WriteSyncer
	if config.DisplayInConsole {
		syncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(hook))
	} else {
		syncer = zapcore.AddSync(hook)
	}

	return syncer
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
