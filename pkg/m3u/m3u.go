package m3u

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/download"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/task"
	"github.com/vbauerster/mpb/v8"
	"go.uber.org/zap"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

/**
1. download a m3u from file
2. read m3u and parser

*/

const jobType = "m3u"

func init() {
	task.NewTaskMap[jobType] = NewM3uTaskCache
}

type Task struct {
	ctx     context.Context
	cancel  context.CancelFunc
	status  atomic.Int32
	tasks   task.Manager
	Url     *url.URL
	Dir     string
	display *display.Display
	cfg     task.Config
}

func (t *Task) GetType() string {
	return jobType
}

func (t *Task) GetStatus() task.Status {
	return t.status.Load()
}

func (t *Task) Stop() error {
	t.status.Store(task.Paused)
	return nil
}

func (t *Task) Resume() error {
	t.status.Store(task.Running)
	return nil
}

func (t *Task) Exit() error {
	t.cancel()
	return nil
}

func (t *Task) Extra() ([]byte, error) {

	return nil, nil
}

func NewM3uTaskCache(ctx context.Context, displayOpt *display.Display, cfg *task.Config, value []byte) (task.Task, error) {
	return &Task{}, nil
}

func NewM3uTask(ctx context.Context, displayOpt *display.Display, cfg *task.Config, url *url.URL, dir string) task.Task {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				log.Error("create dir failed", zap.Error(err), zap.String("dir", dir))
				return nil
			}
		} else {
			log.Error("check dir failed", zap.Error(err), zap.String("dir", dir))
			return nil
		}
	}
	status := filepath.Join(dir, "m3u.cache")

	ctx, cancel := context.WithCancel(ctx)
	t := &Task{
		ctx:     ctx,
		cancel:  cancel,
		status:  atomic.Int32{},
		tasks:   task.NewManager(ctx, cfg.ThreadSize, status),
		Url:     url,
		Dir:     dir,
		display: displayOpt,
		cfg:     *cfg,
	}
	t.status.Store(task.Pending)
	return t
}

func (t *Task) Start() error {
	var err error
	defer func() {
		if t.status.Load() == task.Running {
			if err != nil {
				t.status.Store(task.Aborted)
			} else {
				t.status.Store(task.Completed)
			}
		}
	}()
	t.status.Store(task.Running)
	err = t.run()
	return err
}

func (t *Task) run() error {

	filename := filepath.Base(t.Url.Path)
	m3uJob := download.NewHttpTask(t.ctx, t.Url, filename, true, &t.cfg, t.display)

	err := m3uJob.Start()
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error("download failed", zap.Error(err), zap.String("url", t.Url.String()))
		}
		return err
	}
	log.Info("download success", zap.String("url", t.Url.String()))
	info, err := m3uJob.Extra()
	if err != nil {
		log.Error("get Extra from m3uJob ", zap.Error(err))
		return err
	}
	var jobInfo download.JobInfo
	err = json.Unmarshal(info, &jobInfo)
	if err != nil {
		log.Error("unmarshal m3uJob info failed", zap.Error(err))
	}
	log.Info("start to check", zap.String("file", jobInfo.FileName))

	taskList, err := CheckIsM3u(jobInfo.FileName)
	if err != nil {
		log.Warn("it is not a m3u file", zap.Error(err))
		return err
	}

	// 分割路径，找到最后一个斜杠的位置
	pathParts := strings.Split(t.Url.Path, "/")
	lastPartIndex := len(pathParts) - 1
	newUrl := t.Url
	defer t.tasks.Close()

	var bar *mpb.Bar
	if t.display != nil {
		bar = t.display.AddBar(t.Dir, int64(len(taskList)), "down")
	}
	curr := time.Now()
	for _, jobName := range taskList {
		pathParts[lastPartIndex] = jobName
		newPath := strings.Join(pathParts, "/")
		newUrl.Path = newPath

		file := filepath.Join(t.Dir, jobName)
		job := download.NewHttpTask(t.ctx, newUrl, file, true, &t.cfg, nil)

		err = t.tasks.NewTask(jobName, job)
		if err != nil {
			log.Error("add new job failed", zap.Error(err))
		}
		dur := time.Since(curr)
		display.InCr(bar, 1, dur)
		curr = time.Now()
	}
	return nil
}

func CheckIsM3u(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Error("check is m3u , open failed ", zap.Error(err))
		return nil, err
	}
	defer file.Close()

	// 创建一个新的 Scanner 并关联到文件
	scanner := bufio.NewScanner(file)

	// 读取文件的第一行
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#EXTM3U") {
			return nil, errors.New("it is not a m3u file")
		}
	}
	list := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查是否是媒体文件的 URI 行
		if !strings.HasPrefix(line, "#") {
			list = append(list, line)
		}
		if strings.HasPrefix(line, "#EXT-X-KEY:") {
			ts := GetKey(line)
			list = append(list, ts)
		}
	}
	return list, nil
}

func GetKey(keyLine string) string {
	// #EXT-X-KEY:METHOD=AES-128,URI="ac192406713e606f.ts",IV=0x3b9d6e07420b308025d11a53692d8f51
	parts := strings.Split(keyLine, ",")
	var uri string
	for _, part := range parts {
		if strings.HasPrefix(part, "URI=") {
			// 去掉前缀 URI=" 和结尾的 "
			uri = strings.TrimPrefix(part, "URI=\"")
			uri = strings.TrimSuffix(uri, "\"")

		}
	}
	return uri
}
