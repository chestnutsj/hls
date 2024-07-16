package task

import (
	"context"
	"github.com/chestnutsj/hls/pkg/log"
	"sync/atomic"
	"testing"
	"time"
)

type testJob struct {
	status atomic.Int32
	f      func()
}

func (t *testJob) GetType() string {
	//TODO implement me
	return "testjob"
}

func (t *testJob) Start() error {
	defer t.status.Store(Completed)
	t.status.Store(Running)
	<-time.After(time.Second)
	if t.f != nil {
		t.f()
	}
	return nil
}
func (t *testJob) Stop() error {
	t.status.Store(Paused)
	return nil
}
func (t *testJob) Resume() error {
	t.status.Store(Running)
	return nil
}
func (t *testJob) Exit() error {
	return nil
}
func (t *testJob) GetStatus() Status {
	return t.status.Load()
}
func (t *testJob) Extra() ([]byte, error) { return nil, nil }

func newTestJob(f func()) Task {

	j := &testJob{f: f}
	j.status.Store(Pending)
	return j
}

func Test_taskMgr(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	mgr := NewManager(ctx, 5, "test.xz3")

	if mgr == nil {
		t.Fatal("init mgr failed")
	}
	l := make([]int, 0)
	f := func() {
		t.Log("x")
		l = append(l, 1)
	}
	x := time.Now()

	mgr.NewTask("test1", newTestJob(f))
	mgr.NewTask("test2", newTestJob(f))
	mgr.NewTask("test3", newTestJob(f))
	mgr.NewTask("test4", newTestJob(f))
	mgr.NewTask("test5", newTestJob(f))
	mgr.NewTask("test6", newTestJob(f))
	mgr.NewTask("test7", newTestJob(f))

	mgr.Close()

	wait := time.Since(x)
	if len(l) != 7 {
		t.Fatal("task not run", len(l))
	}
	if wait < (time.Second * 2) {
		t.Fatal("not ctrl task")
	}
}
