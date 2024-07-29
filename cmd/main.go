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
	Commit    = "unknown"
	BuildTime = "unknown"
	Version   = "unknown"
)

type TaskInfo struct {
	TaskKey  url.URL
	FileName string
}

var Cfg = struct {
	Download task.Config `yaml:"download"`
	Log      log.Config  `yaml:"log" default:"warning"`
	Metric   string      `yaml:"metric"`
	Debug    bool        `yaml:"debug" default:"true"`
}{}

func main() {

	configFile := flag.String("config", "", "configuration file")
	urlStr := flag.String("u", "", "download url")
	output := flag.String("o", "", "output dir")
	fileType := flag.Bool("m", false, "it is a m3u8 file")
	version := flag.Bool("v", false, "Show version")
	help := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *help || flag.NArg() == 0 && flag.NFlag() == 0 {
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Println("version:", Version, "build time:", BuildTime, "git commit:", Commit)
		return
	}

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
		log.Error("url is empty")
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
		filename := *output
		if len(*output) == 0 {
			filename = filepath.Base(u.Path)
		}
		job = download.NewHttpTask(ctx, u, filename, false, &Cfg.Download, p)
	}

	err = job.Start()
	if err != nil {
		log.Error("download", zap.Error(err))
	} else {
		fmt.Println("download success ", *urlStr)
	}
}
