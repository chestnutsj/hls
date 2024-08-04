package hook

import "net/rpc"

// PluginRPC 是一个适配器，用于通过 RPC 调用插件方法
type PluginRPC struct{ client *rpc.Client }

func (p *PluginRPC) StartDecoder(info string) error {
	var resp interface{}

	err := p.client.Call("Plugin.StartDecoder", info, &resp)
	if err != nil {
		return err
	}

	return nil
}

// PluginRPCServer 是一个适配器，用于将插件方法暴露为 RPC
type PluginRPCServer struct {
	Impl MyDecoder
}

func (s *PluginRPCServer) StartDecoder(info string, resp *interface{}) error {
	err := s.Impl.StartDecoder(info)
	return err
}
