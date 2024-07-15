package main

import (
	"flag"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/metrics"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
	"net/url"
	"os"
)

var (
	GitCommitLog   = "unknown"
	BuildTime      = "unknown"
	BuildGoVersion = "unknown"
)

type TaskInfo struct {
	TaskKey  url.URL
	FileName string
}

type DownloadConfig struct {
	DownloadDir string `yaml:"download_dir" default:"./downloads"`
	ConnTimeout uint   `yaml:"conn_timeout" default:"5"`
	ChunkSize   uint64 `yaml:"chunk_size" default:"1048576"`
	RetryCount  uint   `yaml:"retry_count" default:"5"`
}

var Cfg = struct {
	Download DownloadConfig `yaml:"download"`
	Log      log.LogConfig  `yaml:"log"`
	Metric   string         `yaml:"metric"`
	Debug    bool           `yaml:"debug" default:"true"`
}{}

func main() {

	configFile := flag.String("config", "", "configuration file")

	flag.Parse()
	_ = os.Setenv("CONFIGOR_ENV_PREFIX", "-")

	var err error
	deamon := false
	if *configFile == "" {
		err = configor.Load(&Cfg)
	} else {
		err = configor.Load(&Cfg, *configFile)
	}

	log.InitLogger(Cfg.Log)
	if err != nil {
		zap.L().Error("load config error", zap.Error(err))
		os.Exit(1)
	}
	if deamon {
		metrics.StartMetrics(Cfg.Metric, Cfg.Debug)
	}
}
