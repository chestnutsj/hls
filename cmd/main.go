package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/download"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/m3u"
	"github.com/chestnutsj/hls/pkg/metrics"
	"github.com/chestnutsj/hls/pkg/task"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
	"net/url"
	"os"
	"path/filepath"
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

var Cfg = struct {
	Download task.Config `yaml:"download"`
	Log      log.Config  `yaml:"log"`
	Metric   string      `yaml:"metric"`
	Debug    bool        `yaml:"debug" default:"true"`
}{}

func main() {

	configFile := flag.String("config", "", "configuration file")
	urlStr := flag.String("url", "", "download url")
	output := flag.String("output", "", "output dir")
	fileType := flag.Bool("m3u", false, "it is a m3u8 file")
	flag.Parse()
	_ = os.Setenv("CONFIGOR_ENV_PREFIX", "-")

	var err error

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

	if len(*urlStr) == 0 {
		return
	}
	u, err := url.Parse(*urlStr)
	if err != nil {
		zap.L().Error("parse url error", zap.Error(err))
		return
	}

	if Cfg.Metric != "" {
		go metrics.StartMetrics(Cfg.Metric, Cfg.Debug)
	}

	ctx := context.Background()
	p := display.NewDisplay()
	var job task.Task
	if *fileType {
		// output dir
		dir := filepath.Dir(*output)

		job = m3u.NewM3uTask(ctx, p, &Cfg.Download, u, dir)
	} else {
		job = download.NewHttpTask(ctx, u, *output, false, &Cfg.Download, p)
	}

	err = job.Start()
	if err != nil {
		log.Error("download", zap.Error(err))
	} else {
		fmt.Println("download success ", urlStr)
	}
}
