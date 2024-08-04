package hook

import (
	"github.com/hashicorp/go-plugin"
	"net/rpc"
)

var PluginName string = "decoder"
var PluginHandshakeCfg = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

var PluginMap = map[string]plugin.Plugin{
	PluginName: &DecoderPlugin{},
}

type MyDecoder interface {
	StartDecoder(string) error
}

type DecoderPlugin struct {
	// Impl Injection
	Impl MyDecoder
}

func (p *DecoderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

func (p *DecoderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPC{client: c}, nil
}
