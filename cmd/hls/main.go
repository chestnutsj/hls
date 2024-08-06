package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/download"
	"github.com/chestnutsj/hls/pkg/hook"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/m3u"
	"github.com/chestnutsj/hls/pkg/metrics"
	"github.com/chestnutsj/hls/pkg/task"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	BuildTime = "unknown"
	Version   = "unknown"
)

var Cfg = struct {
	Download task.Config `yaml:"download"`
	Log      log.Config  `yaml:"log"`
	Metric   string      `yaml:"metric"`
	Debug    bool        `yaml:"debug" default:"true"`
}{}

func main() {

	configFile := flag.String("config", "", "configuration file")
	urlStr := flag.String("u", "", "download url")
	output := flag.String("o", "", "output dir")
	m3uUrl := flag.String("m", "", "it is a m3u8 file")
	version := flag.Bool("v", false, "Show version")
	help := flag.Bool("h", false, "Show help")
	loadPlugin := flag.String("plugin", hook.PluginName, "download decode plugin")
	genCfg := flag.Bool("genCfg", false, "generator config ")

	flag.Parse()

	if *help || flag.NArg() == 0 && flag.NFlag() == 0 {
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Println("version:", Version, "build time:", BuildTime)
		return
	}
	if *genCfg {
		fmt.Println("cfg demo:")
		data, err := yaml.Marshal(&Cfg)
		if err == nil {
		}
		fmt.Println(string(data))
		return
	}

	_ = os.Setenv("CONFIGOR_ENV_PREFIX", "-")

	var err error

	if *configFile == "" {
		err = configor.Load(&Cfg)
		if os.Getenv("THREAD_SIZE") != "" {
			fmt.Println("THREAD_SIZE", os.Getenv("THREAD_SIZE"))
			s, err := strconv.Atoi(os.Getenv("THREAD_SIZE"))
			if err == nil {
				Cfg.Download.ThreadSize = s
			}
		}
	} else {
		err = configor.Load(&Cfg, *configFile)
	}
	fmt.Println("config:", Cfg)

	if len(*m3uUrl) > 0 {
		fmt.Println("url", *m3uUrl)
	} else {
		fmt.Println("url", *urlStr)
	}
	fmt.Println("output", *output)

	log.InitLogger(Cfg.Log)
	if err != nil {
		zap.L().Error("load config error", zap.Error(err))
		os.Exit(1)
	}

	if Cfg.Metric != "" {
		go metrics.StartMetrics(Cfg.Metric, Cfg.Debug)
	}

	ctx := context.Background()
	p := display.NewDisplay()
	var job task.Task
	if len(*m3uUrl) > 0 {
		u, err := url.Parse(*m3uUrl)
		if err != nil {
			zap.L().Error("parse url error", zap.Error(err))
			return
		}
		dir := *output
		if len(dir) <= 2 {
			// output dir
			dir := filepath.Dir(*output)
			if dir == "." {
				dir = filepath.Base(u.Path)
			}
		}
		job = m3u.NewM3uTask(ctx, p, &Cfg.Download, u, dir)
	} else {

		if len(*urlStr) == 0 {
			log.Error("url is empty")
			return
		}
		u, err := url.Parse(*urlStr)
		if err != nil {
			zap.L().Error("parse url error", zap.Error(err))
			return
		}

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
		log.Info("download success")
		fmt.Println("download success ", *urlStr)

		if len(*m3uUrl) > 0 && len(*loadPlugin) > 0 {
			data, err := job.Extra()
			if err == nil {
				info := make(map[string]interface{})
				err = json.Unmarshal(data, &info)
				if err == nil {

					pluginNameAbs, err := filepath.Abs(*loadPlugin)
					if err != nil {
						zap.L().Error("get pluginNameFile abs failed", zap.Error(err))
						return
					}
					pluginName := filepath.Base(pluginNameAbs)
					pluginNameAbs = addExeSuffix(pluginNameAbs)
					hook.LoadPlugin(pluginNameAbs, pluginName, info)

				}
			}
		}
	}

}

func addExeSuffix(n string) string {

	if len(filepath.Ext(n)) == 0 {
		isWindows := strings.Contains(os.Getenv("GOOS"), "windows")
		if isWindows {
			n += ".exe"
		}

	}

	return n
}
