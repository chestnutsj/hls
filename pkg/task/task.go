package task

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/store"
	"go.uber.org/zap"
	"os"
	"sync"

	"github.com/chestnutsj/hls/pkg/tools"
)

type Status = int32

const (
	Pending Status = iota
	Running
	Paused
	Completed

	Aborted
)

var statusMap = map[Status]string{
	Pending:   "pending",
	Running:   "running",
	Paused:    "paused",
	Completed: "completed",
	Aborted:   "aborted",
}

type Config struct {
	ConnTimeout uint              `yaml:"conn_timeout" env:"CONN_TIMEOUT" default:"10"`
	ChunkSize   int64             `yaml:"chunk_size" default:"1048576"`
	RetryCount  uint              `yaml:"retry_count" env:"RETRY" default:"10"`
	ThreadSize  int               `yaml:"thread_size" env:"THREAD_SIZE" default:"10"`
	Headers     map[string]string `yaml:"headers"`
}

func NewDownloadConfig() *Config {
	return &Config{
		ConnTimeout: 5,
		ChunkSize:   1024 * 1024 * 10,
		RetryCount:  5,
		ThreadSize:  10,
	}
}

type Task interface {
	GetStatus() Status
	Start() error
	Stop() error
	Resume() error
	Exit() error
	Extra() ([]byte, error)
	GetType() string
}
type NewTask func(ctx context.Context, displayOpt *display.Display, cfg *Config, value []byte) (Task, error)

var (
	NewTaskMap = make(map[string]NewTask)
)

type Manager interface {
	NewTask(name string, t Task) error
	GetTask(name string) (Task, error)
	ExitTask(name string) error
	StopTask(name string) error
	ResumeTask(name string) error
	Close() error
	StopAll() error
	ResumeAll() error
	GetAll() ([]interface{}, error)
	Resize(newMaxWorkers int)
}

type manager struct {
	tasks  tools.OrderMap
	ctx    context.Context
	cancel context.CancelFunc

	stop   chan []interface{}
	resume chan []interface{}

	cache *store.BitCask

	works      chan *worker   // 任务通道
	wg         sync.WaitGroup // 同步等待组
	maxWorkers int            // 最大工作数量
	mutex      sync.Mutex     // 保护 maxWorkers 读写的互斥锁

}

type worker struct {
	key string
	t   Task
}
type WorkInfo struct {
	Status Status
	Extra  []byte
	Type   string
}

func (w *worker) SaveInCache(cache *store.BitCask) {
	if cache != nil {
		data, err := w.t.Extra()
		if err == nil {
			info := WorkInfo{
				Extra:  data,
				Type:   w.t.GetType(),
				Status: w.t.GetStatus(),
			}
			b, _ := json.Marshal(info)
			_ = cache.Set(w.key, b)
		}

	}
}

func (m *manager) workRun(name string, t Task) error {
	defer m.tasks.Delete(name)
	return t.Start()
}

func (m *manager) NewTask(name string, t Task) error {
	_, exists := m.tasks.Get(name)
	if exists {
		return fmt.Errorf("task %s already exists", name)
	}
	m.tasks.Set(name, t)
	w := &worker{key: name, t: t}
	if m.cache != nil {
		w.SaveInCache(m.cache)
	}
	m.works <- w
	return nil
}

func (m *manager) GetTask(name string) (Task, error) {
	t, e := m.tasks.Get(name)
	if !e {
		return nil, nil
	}
	return t.(Task), nil
}

func (m *manager) ExitTask(name string) error {
	t, e := m.tasks.Get(name)
	if !e {
		return nil
	}
	return t.(Task).Exit()
}

func (m *manager) StopTask(name string) error {
	t, e := m.tasks.Get(name)
	if !e {
		return nil
	}
	return t.(Task).Stop()
}

func (m *manager) ResumeTask(name string) error {
	t, e := m.tasks.Get(name)
	if !e {
		return nil
	}
	return t.(Task).Resume()
}

func (m *manager) Close() error {
	close(m.works)
	m.wg.Wait()
	if m.cache != nil {
		return m.cleanCache()
	}
	return nil
}

func (m *manager) StopAll() error {
	err := m.tasks.Fetch(func(i interface{}) error {
		return i.(Task).Stop()
	}, true)
	close(m.stop)
	return err
}

func (m *manager) ResumeAll() error {
	m.stop = make(chan []interface{})
	close(m.resume)
	m.resume = make(chan []interface{})
	err := m.tasks.Fetch(func(i interface{}) error {
		return i.(Task).Resume()
	}, true)
	return err
}

func (m *manager) GetAll() ([]interface{}, error) {
	return m.tasks.Values(), nil
}

func (m *manager) cleanCache() error {
	if m.cache != nil {
		stop := false

		err := m.cache.Fetch(func(key string, value []byte) bool {
			var workInfo WorkInfo
			err := json.Unmarshal(value, &workInfo)
			if err == nil {
				if workInfo.Status != Completed {
					stop = true
					return true
				}
			}
			return false
		})
		_ = m.cache.Close()
		if err != nil {
			return err
		}
		if !stop {
			_ = os.RemoveAll(m.cache.GetPath())
		}
	}
	return nil
}

func NewManager(ctx context.Context, maxWorkers int, status string) Manager {
	cache, err := store.NewBitCask(status)
	if err != nil {
		log.Error("create mgr status failed", zap.String("name", status), zap.Error(err))
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	m := &manager{
		tasks:  tools.NewOrderedMap(),
		ctx:    ctx,
		cancel: cancel,

		stop:       make(chan []interface{}),
		resume:     make(chan []interface{}),
		cache:      cache,
		maxWorkers: maxWorkers,
		works:      make(chan *worker),
	}
	go m.run()
	return m
}

// run 运行工作池的 Goroutines
func (m *manager) run() {
	for i := 0; i < m.maxWorkers; i++ {
		m.wg.Add(1)
		go m.worker()
	}
}

func (m *manager) worker() {
	defer m.wg.Done()
	for t := range m.works {
		if t != nil {
			err := m.workRun(t.key, t.t)
			if err != nil {
				log.Error("work run failed", zap.Error(err))
			}
			if m.cache != nil {
				t.SaveInCache(m.cache)
			}
		} else {
			return
		}
	}
}

func (m *manager) Resize(newMaxWorkers int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if newMaxWorkers <= 0 {
		newMaxWorkers = 1
	}

	if newMaxWorkers == m.maxWorkers {
		return
	}

	// 如果增加了工作数量，则增加新的 Goroutines
	if newMaxWorkers > m.maxWorkers {
		for i := m.maxWorkers; i < newMaxWorkers; i++ {
			m.wg.Add(1)
			go m.worker()
		}
	} else {
		// 减少工作数量，关闭多余的 Goroutines
		for i := newMaxWorkers; i < m.maxWorkers; i++ {
			m.works <- nil
		}
	}

	m.maxWorkers = newMaxWorkers
}
