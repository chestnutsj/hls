package hook

import (
	"encoding/json"
	"fmt"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"go.uber.org/zap"
	"os"
	"os/exec"
)

func LoadPlugin(pluginNameAbs string, pluginName string, info map[string]interface{}) {
	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})
	// 创建插件客户端
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  PluginHandshakeCfg,
		Plugins:          PluginMap,
		Cmd:              exec.Command(pluginNameAbs, "-plugin", pluginName),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
		Logger:           logger,
	})

	defer client.Kill()

	fmt.Println(pluginNameAbs)
	// 启动插件客户端
	rpcClient, err := client.Client()
	if err != nil {
		log.Error("start plugin failed", zap.Error(err))
		return
	}

	log.Info("start dispense")
	// 请求插件
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		log.Error("dispense plugin failed", zap.Error(err))
		return
	}
	dec, ok := raw.(MyDecoder)
	if !ok {
		log.Error("can't Dispense")
		return
	}

	data, err := json.Marshal(info)
	if err != nil {
		log.Error("marshal info failed", zap.Error(err))
		return
	}

	err = dec.StartDecoder(string(data))
	if err != nil {
		zap.L().Error("run decoder fail", zap.Error(err))
	}

}
