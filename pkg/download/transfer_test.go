package download

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/task"
	"go.uber.org/zap"
)

func Test_ser(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}

	dataLen := 10000
	data := []byte(textGenerator(dataLen))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Debug("download", zap.String("url", r.URL.String()))
		// 设置 Content-Disposition header 为 attachment，这样浏览器会提示用户下载文件
		w.Header().Set("Content-Disposition", "attachment; filename=example.txt")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		x, err := w.Write(data)
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		if x != len(data) {
			zap.L().Error("send data not", zap.Int("send", x), zap.Int("dataLen", len(data)))
		}
	}))
	defer ts.Close()
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	x, err := io.ReadAll(resp.Body)

	if !bytes.Equal(data, x) {
		t.Fatal("data is not eq")
	}
}

func Test_transfer(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	dataLen := 1000000
	data := []byte(textGenerator(dataLen))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Debug("download", zap.String("url", r.URL.String()))
		// 设置 Content-Disposition header 为 attachment，这样浏览器会提示用户下载文件
		w.Header().Set("Content-Disposition", "attachment; filename=example.txt")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		x, err := w.Write(data)
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		if x != len(data) {
			zap.L().Error("send data not", zap.Int("send", x), zap.Int("dataLen", len(data)))
		}
	}))
	defer ts.Close()

	status := atomic.Int32{}
	client := NewClient(ctx, 3, time.Second*3, time.Second*30)

	readChan := make(chan FileData, 3)
	rev := make([]byte, 0, len(data))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for rd := range readChan {
			// t.Log("read", rd.Pos, "len", len(rd.Data))

			if data[rd.GetPos()] != rd.GetData()[0] {
				t.Error("data not equal", rd.GetPos(), "recv", string(rd.GetData()[0:10]), "data", string(data[rd.GetPos():rd.GetPos()+10]))
			}

			rev = append(rev, rd.GetData()...)
		}
	}()

	tr := NewTransfer(ctx, &status, client, ts.URL, nil, int64(dataLen/10))
	status.Store(task.Running)
	err = tr.DownloadPerThread(readChan, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	close(readChan)
	wg.Wait()

	if len(data) != len(rev) {
		t.Fatal("data not equal len")
	}

	if !bytes.Equal(data, rev) {
		minLen := min(len(data), len(rev))

		for i := 0; i < minLen; {
			if data[i] != rev[i] {

				t.Log("diff at", i, ":", hex.Dump(data[i:i+10]), "!=", hex.Dump(rev[i:i+10]))
			}
			i += (minLen / 10)
		}
		t.Fatal("data not equal")

	}
}

func Test_transfer_muti(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	dataLen := 1000000
	data := []byte(textGenerator(dataLen))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Debug("download", zap.String("url", r.URL.String()))
		// 设置 Content-Disposition header 为 attachment，这样浏览器会提示用户下载文件
		w.Header().Set("Content-Disposition", "attachment; filename=example.txt")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))

		r.Header.Get("Range")

		x, err := w.Write(data)
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		if x != len(data) {
			zap.L().Error("send data not", zap.Int("send", x), zap.Int("dataLen", len(data)))
		}
	}))
	defer ts.Close()

	status := atomic.Int32{}
	client := NewClient(ctx, 3, time.Second*3, time.Second*30)

	readChan := make(chan FileData, 3)
	rev := make([]byte, 0, len(data))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for rd := range readChan {
			// t.Log("read", rd.Pos, "len", len(rd.Data))

			if data[rd.GetPos()] != rd.GetData()[0] {
				t.Error("data not equal", rd.GetPos(), "recv", string(rd.GetData()[0:10]), "data", string(data[rd.GetPos():rd.GetPos()+10]))
			}

			rev = append(rev, rd.GetData()...)
		}
	}()

	tr := NewTransfer(ctx, &status, client, ts.URL, nil, int64(dataLen/10))
	status.Store(task.Running)
	err = tr.DownloadPerThread(readChan, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	close(readChan)
	wg.Wait()

	if len(data) != len(rev) {
		t.Fatal("data not equal len")
	}

	if !bytes.Equal(data, rev) {
		minLen := min(len(data), len(rev))

		for i := 0; i < minLen; {
			if data[i] != rev[i] {

				t.Log("diff at", i, ":", hex.Dump(data[i:i+10]), "!=", hex.Dump(rev[i:i+10]))
			}
			i += (minLen / 10)
		}
		t.Fatal("data not equal")

	}
}
