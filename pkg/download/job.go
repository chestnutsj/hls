package download

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/task"
	"github.com/chestnutsj/hls/pkg/tools"
	"github.com/vbauerster/mpb/v8"
	"go.uber.org/zap"
)

const JobType = "download"

func init() {
	task.NewTaskMap[JobType] = NewHttpTaskByCache
}

const statusSuffix = ".xz3"

type JobInfo struct {
	Url        string
	FileName   string
	SourceFile string
}

type Job struct {
	sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *task.Config
	info       JobInfo
	client     MyClient
	lastErr    error
	status     atomic.Int32
	jobWg      sync.WaitGroup
	displayOpt *display.Display
}

func NewHttpTaskByCache(ctx context.Context, displayOpt *display.Display, cfg *task.Config, info []byte) (task.Task, error) {
	var i JobInfo
	err := json.Unmarshal(info, &i)
	if err != nil {
		log.Error("recreate failed", zap.Error(err))
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	j := &Job{
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		info:       i,
		client:     NewClient(ctx, int(cfg.RetryCount), time.Duration(cfg.ConnTimeout)*time.Second, time.Duration(cfg.ConnTimeout)*time.Second),
		status:     atomic.Int32{},
		jobWg:      sync.WaitGroup{},
		displayOpt: displayOpt,
	}

	j.status.Store(task.Pending)
	return j, nil
}

func NewHttpTask(ctx context.Context, url *url.URL, filename string, force bool, cfg *task.Config, displayOpt *display.Display) task.Task {
	source := filename

	if !force {
		filename, _ = tools.GenerateUniqueFilename(filename, statusSuffix)
	}
	ctx, cancel := context.WithCancel(ctx)
	j := &Job{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		info: JobInfo{
			Url:        url.String(),
			FileName:   filename,
			SourceFile: source,
		},
		client: NewClient(ctx, int(cfg.RetryCount), time.Duration(cfg.ConnTimeout)*time.Second, time.Duration(cfg.ConnTimeout)*time.Second),

		status:     atomic.Int32{},
		jobWg:      sync.WaitGroup{},
		displayOpt: displayOpt,
	}

	j.status.Store(task.Pending)
	return j
}

func (j *Job) GetType() string {
	return JobType
}

func (j *Job) GetStatus() task.Status {
	return j.status.Load()
}
func (j *Job) Start() error {
	var err error
	defer func() {
		if j.status.Load() == task.Running {
			if j.lastErr == nil || err == nil {
				j.status.Store(task.Completed)

			} else {
				j.status.Store(task.Aborted)
			}
		}
		j.cancel()
	}()
	j.status.Store(task.Running)
	err = j.work()
	j.jobWg.Wait()
	return err
}

func (j *Job) Stop() error {
	j.status.Store(task.Paused)
	return nil
}

func (j *Job) Resume() error {
	j.status.Store(task.Running)
	return nil
}

func (j *Job) Exit() error {
	j.cancel()
	return nil
}

func (j *Job) Extra() ([]byte, error) {
	return json.Marshal(&j.info)
}

func checkRangeSupportAndGetSize(resp *http.Response) (int64, bool, error) {
	acceptRanges := resp.Header.Get("Accept-Ranges")
	supportsRange := acceptRanges == "bytes"

	if !supportsRange {
		log.Info("Server does not support byte ranges")
	}

	contentLengthStr := resp.Header.Get("Content-Length")
	var contentLength int64
	var err error
	if contentLengthStr != "" {
		contentLength, err = strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			return 0, false, err
		}
	}

	return contentLength, supportsRange, nil
}

func (j *Job) work() error {
	urlStr := j.info.Url
	req, err := j.client.NewRequest(urlStr, j.cfg.Headers)
	if err != nil {
		return err
	}
	zap.L().Info("NewRequest", zap.Any("req", req.Header))

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("resp is empty")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("%s resp is %d", urlStr, resp.StatusCode)
	}
	contentLength, supportsRange, err := checkRangeSupportAndGetSize(resp)
	if err != nil {
		zap.L().Info("check resp failed")
		contentLength = 0
		supportsRange = false
	}
	zap.L().Info("start download 200", zap.Int64("contentLength", contentLength), zap.Bool("range", supportsRange))
	var bar *mpb.Bar
	if j.displayOpt != nil {
		bar = j.displayOpt.AddBar(j.info.FileName, int64(contentLength), "down")
	}

	prof := NewProgress(bar)
	defer func() {
		prof.Close()
		if prof.cache != nil {
			if err != nil && j.ctx.Err() == nil {
				_ = os.RemoveAll(prof.cache.GetPath())
			}
		}
	}()

	chunk, err := NewChunk(j.info.FileName, prof)
	if err != nil {
		return err
	}
	j.jobWg.Add(1)
	go func() {
		defer func() {
			_ = chunk.Close()
			j.jobWg.Done()
		}()
		chunk.Run()
	}()
	defer chunk.Exit()
	transfer := NewTransfer(j.ctx, &j.status, j.client, urlStr, j.cfg.Headers, j.cfg.ChunkSize)
	if supportsRange && contentLength > j.cfg.ChunkSize && j.cfg.ThreadSize > 1 {
		zap.L().Info("start download range")
		meta, _ := json.Marshal(j.info)
		err = prof.InitCache(j.info.SourceFile, statusSuffix, meta)
		if err != nil {
			return err
		}
		jobsMap := prof.GetTasks(contentLength, j.cfg.ChunkSize)
		if len(jobsMap) == 0 {
			zap.L().Info("task is download over ")
			return nil
		} else {
			zap.L().Info("reStart download", zap.Int("jobsMap", len(jobsMap)))
		}
		err = transfer.DownloadMtiThread(chunk.writeChan, j.cfg.ThreadSize, jobsMap)

	} else {
		zap.L().Info("start download single")
		err = transfer.DownloadPerThread(chunk.writeChan, 0, 0)
		zap.L().Info("download single exit")
	}

	if j.ctx.Err() != nil {
		zap.L().Warn("context canceled")
		if bar != nil {
			bar.Abort(true)
		}
		j.status.Store(task.Aborted)
		zap.L().Info("download job has cancel")
	} else {
		zap.L().Info("download job has stop")
	}

	return err
}
