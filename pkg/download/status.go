package download

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/store"
	"github.com/chestnutsj/hls/pkg/tools"
	"github.com/vbauerster/mpb/v8"
	"go.uber.org/zap"
)

type Progress struct {
	cache *store.BitCask
	bar   *mpb.Bar
	curr  time.Time
}

func NewProgress(bar *mpb.Bar) *Progress {
	return &Progress{
		cache: nil,
		bar:   bar,
		curr:  time.Now(),
	}
}

func (p *Progress) InitCache(path, ext string, metadata []byte) error {
	path = tools.GetStatusExt(path, ext)
	statusDb, err := store.NewBitCask(path)
	if err != nil {
		zap.L().Error("init status cache failed", zap.Error(err))
		return err
	}
	if statusDb.Len() != 0 {
		cache, err := statusDb.Get("status")
		if err == nil {
			if len(cache) == len(metadata) {
				if bytes.Equal(cache, metadata) {
					p.cache = statusDb
					return nil
				}
			}
		}
		zap.L().Info("restart from cache info")
		err = statusDb.Reset()
		if err != nil {
			zap.L().Error("reset failed", zap.Error(err))
			return err
		}
	}
	p.cache = statusDb

	return statusDb.Set("status", metadata)
}

func (p *Progress) GetTasks(total int64, chunkSize int64) map[int64]int64 {
	if total == 0 {
		return nil
	}

	tasks := make(map[int64]int64)
	tools.AddUncovered(tasks, 0, total, chunkSize)

	l := p.cache.Len()
	if l > 1 {
		orderList := make([]int64, 0, l)
		lastLen := int64(0)

		for start, end := range tasks {
			var key bytes.Buffer
			_ = binary.Write(&key, binary.BigEndian, start)
			v, err := p.cache.Get(key.String())
			if err != nil {
				continue
			}
			length := int64(binary.BigEndian.Uint64(v))
			if (length + start) >= end {
				orderList = append(orderList, start)
				lastLen += length
			}
		}
		zap.L().Info("has already find cache", zap.Int("orderList", len(orderList)))
		for _, k := range orderList {
			delete(tasks, k)
		}
		if p.bar != nil {
			p.bar.SetCurrent(lastLen)
		}
	}
	return tasks
}

func (p *Progress) Close() {

	if p.cache != nil {
		_ = p.cache.Close()
	}
}
func (p *Progress) UpdateStatus(data FileData) {
	if p.cache != nil {
		offset := data.GetStart()
		l := data.GetOffsetLen()
		var key bytes.Buffer
		var val bytes.Buffer
		_ = binary.Write(&key, binary.BigEndian, offset)
		_ = binary.Write(&val, binary.BigEndian, l)
		_ = p.cache.Set(key.String(), val.Bytes())
	}
	if p.bar != nil {
		pos := data.GetDataLen()
		x := time.Since(p.curr)
		display.InCr(p.bar, pos, x)
		p.curr = time.Now()
	}
}
