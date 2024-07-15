package log

import (
	"fmt"
	"github.com/chestnutsj/hls/pkg/tools"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type LogConfig struct {
	Std     bool          `yaml:"std" default:"true"`
	Dir     string        `yaml:"dir" default:"log" `
	Level   zapcore.Level `yaml:"level" default:"info"`
	MaxFile int           `default:"7"`
	MaxAge  int           `default:"1"`
}

func InitLogger(cfg LogConfig) {
	app := tools.AppName()
	file := fmt.Sprintf("%s/%s.log", cfg.Dir, app) //filePath
	hook := lumberjack.Logger{
		Filename:   file,
		MaxBackups: cfg.MaxFile,
		MaxAge:     cfg.MaxAge, //days
		Compress:   true,       // disabled by default
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "file",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			type appendTimeEncoder interface {
				AppendTimeLayout(time.Time, string)
			}
			if enc, ok := enc.(appendTimeEncoder); ok {
				enc.AppendTimeLayout(t, "2006-01-02 15:04:05.000")
				return
			}
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		},
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	level := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= cfg.Level
	})
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return cfg.Std
	})

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(&hook)), // 打印到控制台和文件
			level),

		zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), lowPriority),
	)
	Logger := zap.New(core, zap.AddCaller())

	//Logger, _ = zap.NewProduction()

	defer func() {
		_ = Logger.Sync()
	}()

	zap.ReplaceGlobals(Logger)
	zap.RedirectStdLog(Logger)
	Logger.Info("logger start")
}
