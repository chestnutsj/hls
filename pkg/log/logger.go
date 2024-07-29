package log

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/chestnutsj/hls/pkg/tools"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Std     bool          `yaml:"std" default:"true"`
	Dir     string        `yaml:"dir" default:"log" `
	Level   zapcore.Level `yaml:"level" default:"info"`
	MaxFile int           `default:"7"`
	MaxAge  int           `default:"1"`
}

var (
	once   sync.Once
	devErr error
	level  zap.AtomicLevel
	logger *zap.Logger
)

func init() {
	level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
}

func SetLogLevel(l string) {
	err := level.UnmarshalText([]byte(l))
	if err != nil {
		return
	}
}

func Debug(msg string, fields ...zap.Field) {
	if logger != nil {
		logger.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...zap.Field) {
	if logger != nil {
		logger.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...zap.Field) {
	if logger != nil {
		logger.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if logger != nil {
		logger.Error(msg, fields...)
	}
}
func DevLog() error {
	var err error
	once.Do(func() {
		err = devLog()
	})
	return err
}

func devLog() error {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.Level = level
	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(logger)
	return nil
}

func InitLogger(cfg Config) {
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

	level.SetLevel(cfg.Level)
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
	logger = zap.New(core, zap.AddCaller())

	//Logger, _ = zap.NewProduction()

	defer func() {
		_ = logger.Sync()
	}()

	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)
	logger.Info("logger start")
}

func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}
