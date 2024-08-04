package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/chestnutsj/hls/pkg/hook"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"os"
	"path/filepath"
)

var (
	BuildTime = "unknown"
	Version   = "unknown"
)

type Decoder struct {
}

func NewDecoder() hook.MyDecoder {
	return &Decoder{}
}
func (Decoder) StartDecoder(data string) error {

	info := make(map[string]interface{})
	err := json.Unmarshal([]byte(data), &info)
	if err != nil {
		return err
	}
	fileName, ok := info["file"].(string)
	if !ok {
		return errors.New("can't get url ")
	}

	name := filepath.Base(fileName)

	dirName := filepath.Dir(fileName)

	dir := filepath.Base(dirName)

	shellName := filepath.Join(dirName, "decode.sh")
	file, err := os.OpenFile(shellName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("can't open file %s", shellName)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("#!/bin/bash\n"))
	if err != nil {

		return err
	}
	_, err = file.WriteString(fmt.Sprintf("cd %s\n", filepath.ToSlash(dirName)))
	if err != nil {
		return err
	}
	_, err = file.WriteString(fmt.Sprintf("ffmpeg  -allowed_extensions ALL -i '%s' -c:v h264_nvenc '%s'.mp4\n", name, dir))
	if err != nil {
		return err
	}
	return nil
}

func main() {
	loadPlugin := flag.String("plugin", hook.PluginName, "download decode plugin")
	help := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *help || flag.NArg() == 0 && flag.NFlag() == 0 || len(*loadPlugin) == 0 {
		flag.PrintDefaults()
		return
	}
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})
	logger.Debug("message from decoder")
	dec := NewDecoder()
	cfg := plugin.ServeConfig{
		HandshakeConfig: hook.PluginHandshakeCfg,
		Plugins: map[string]plugin.Plugin{
			*loadPlugin: &hook.DecoderPlugin{Impl: dec},
		},
		GRPCServer: nil,
	}
	plugin.Serve(&cfg)

}
