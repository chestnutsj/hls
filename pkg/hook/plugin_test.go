package hook

import (
	"github.com/chestnutsj/hls/pkg/log"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
)

func Test_plugin(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}

	info := map[string]interface{}{
		"file": "download/xxxxx/29986.m3u8",
	}
	err = os.MkdirAll("download/xxxxx", os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("download/")
	pluginNameFile := "C:\\work\\hls\\output\\decoder.exe"

	pluginNameAbs, err := filepath.Abs(pluginNameFile)
	if err != nil {
		zap.L().Error("get pluginNameFile abs failed", zap.Error(err))
		return
	}
	//	pluginName := filepath.Base(pluginNameAbs)
	_, err = os.Stat(pluginNameAbs)
	if err != nil {
		t.Fatal(err)
	}

	LoadPlugin(pluginNameAbs, "decoder", info)
}
