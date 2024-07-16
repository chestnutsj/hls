package download

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/chestnutsj/hls/pkg/tools"
	"go.uber.org/zap"
)

type MyClient interface {
	Cancel()
	Do(req *http.Request) (*http.Response, error)
	NewRequest(url string, headers map[string]string) (*http.Request, error)
}

type myClient struct {
	client     *http.Client
	maxRetries int
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewClient(ctx context.Context, maxRetry int, connTimeOut time.Duration, idleTimeOut time.Duration) MyClient {
	ctx, cancel := context.WithCancel(ctx)

	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: connTimeOut}).DialContext,
		MaxIdleConnsPerHost: 100,
		MaxIdleConns:        100,
		IdleConnTimeout:     idleTimeOut,
	}
	return &myClient{
		client: &http.Client{
			Transport: transport,
		},
		maxRetries: maxRetry,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (c *myClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	retries := 0
	req = req.WithContext(c.ctx)

	for retries <= c.maxRetries {
		resp, err = c.client.Do(req)
		if err == nil {
			return resp, nil
		}
		zap.L().Debug("retrying", zap.Int("retries", retries), zap.Error(err))
		if c.maxRetries >= 0 {
			retries++
		}
		select {
		case <-c.ctx.Done():
			return nil, nil
		case <-time.After(time.Second):
		}
	}
	zap.L().Error("connect failed", zap.Error(err))
	return nil, err
}

func (c *myClient) NewRequest(url string, headerCfg map[string]string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		zap.L().Error("new req failed", zap.Error(err))
		return nil, err
	}

	headers := tools.NewCaseInsensitiveMap()
	for k, v := range headerCfg {
		headers.Set(k, v)
	}

	_, exist := headers.Get("User-Agent")
	if !exist {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	}
	headers.Range(func(key string, value interface{}) bool {
		req.Header.Set(key, value.(string))
		zap.L().Debug("set header", zap.String("key", key), zap.String("value", value.(string)))
		return false
	})
	return req, nil
}

func (c *myClient) Cancel() {
	c.cancel()
}
