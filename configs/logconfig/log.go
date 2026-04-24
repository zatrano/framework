package logconfig

import (
	"strings"

	"github.com/zatrano/framework/configs/envconfig"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger
var SLog *zap.SugaredLogger

func InitLogger() {
	if Log != nil {
		return
	}

	isProd := envconfig.IsProd()
	var cfg zap.Config
	var lvl zapcore.Level

	if isProd {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "console" // sade metin
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // renkli
		cfg.EncoderConfig.EncodeCaller = nil                             // caller bilgisini kapat
		lvl = zapcore.InfoLevel
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // renkli
		cfg.EncoderConfig.EncodeCaller = nil
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		lvl = zapcore.DebugLevel
	}

	// LOG_LEVEL override
	if v := strings.TrimSpace(envconfig.String("LOG_LEVEL", "")); v != "" {
		if parsed, err := zapcore.ParseLevel(v); err == nil {
			lvl = parsed
		} else {
			tmp, _ := zap.NewDevelopment()
			tmp.Sugar().Warnw("Invalid LOG_LEVEL; using default",
				"value", v, "default", lvl.String(), "error", err)
			_ = tmp.Sync()
		}
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	// Günlükleyici oluştur
	l, err := cfg.Build()
	if err != nil {
		panic("Zap logger başlatılamadı: " + err.Error())
	}

	Log = l
	SLog = Log.Sugar()

	SLog.Infow("Zap logger başarıyla başlatıldı",
		"environment", tern(isProd, "production", "development"),
		"log_level", cfg.Level.Level().String(),
	)
}

func SyncLogger() {
	if Log != nil {
		_ = Log.Sync()
	}
	if SLog != nil {
		_ = SLog.Sync()
	}
}

// küçük yardımcı
func tern[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
