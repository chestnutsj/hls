package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chestnutsj/hls/pkg/task"
	"go.uber.org/zap"
)

type Transfer struct {
	ctx      context.Context
	cancel   context.CancelFunc
	client   MyClient
	url      string
	buffSize int64
	status   *atomic.Int32
	header   map[string]string
}

func NewTransfer(ctx context.Context, status *atomic.Int32, client MyClient, url string, header map[string]string, bufSize int64) *Transfer {
	ctx, cancel := context.WithCancel(ctx)
	return &Transfer{
		ctx:      ctx,
		cancel:   cancel,
		client:   client,
		url:      url,
		buffSize: bufSize,
		header:   header,
		status:   status,
	}
}

func (t *Transfer) DownloadPerThread(write chan FileData, start, end int64) error {
	ctx, cancel := context.WithCancel(t.ctx)
	defer cancel()

	req, err := t.client.NewRequest(t.url, t.header)
	if err != nil {
		return err
	}
	if end != 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
		zap.L().Debug("download", zap.Int64("start", start), zap.Int64("end", end))
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return fmt.Errorf("%s resp is failed code:%s", t.url, resp.Status)
	}
	// zap.L().Debug("resp", zap.Any("head", resp.Header))

	buffer := make([]byte, t.buffSize)
	offset := start
	var n int
	for {
		select {
		case <-ctx.Done():
			{
				return nil
			}
		default:
			if t.status.Load() == task.Running {
				n, err = resp.Body.Read(buffer)
				if err != nil && err != io.EOF {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					zap.L().Error("read failed", zap.Error(err))
					return err
				}
				if n > 0 {
					f := NewFileData(offset, buffer[:n], start)
					select {
					case <-ctx.Done():
						zap.L().Warn("context cancel")
						return nil
					case write <- f:
						{
							offset += int64(n)
						}
					}
				}
				if err == io.EOF {
					return nil
				}
				if end != 0 && offset > end {
					return nil
				}
			} else {
				<-time.After(time.Second)
			}
		}
	}
}

func (t *Transfer) DownloadMtiThread(write chan FileData, ThreadSize int, ms map[int64]int64) error {
	wg := sync.WaitGroup{}
	limit := make(chan struct{}, ThreadSize)
	var err error

	for k, v := range ms {
		wg.Add(1)
		go func(start, end int64) {
			defer func() {
				<-limit
				wg.Done()
			}()
			limit <- struct{}{}
			if t.ctx.Err() != nil {
				return
			}
			xErr := t.DownloadPerThread(write, start, end)
			if xErr != nil {
				if !errors.Is(xErr, context.Canceled) {
					zap.L().Error("DownloadPerThread failed", zap.Error(xErr))
				}
				err = xErr
			}
		}(k, v)
	}
	wg.Wait()
	zap.L().Info("DownloadMtiThread exit")
	return err
}
