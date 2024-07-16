package display

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vbauerster/mpb/v8"
)

func TestAddAfterDone(t *testing.T) {
	p := mpb.New(mpb.WithOutput(os.Stdout))
	bar := p.AddBar(100)
	bar.IncrBy(100)

	p.Wait()

	_, err := p.Add(100, nil)

	if err != mpb.DoneError {
		t.Errorf("Expected %q, got: %q\n", mpb.DoneError, err)
	}
}

func Test_speed(t *testing.T) {
	if !testing.Verbose() {
		t.Skip("Skipping progress bars test in non-verbose mode")
	}
	if Disable {
		t.Skip("Skipping progress bars test")
	}

	// passed wg will be accounted at p.Wait() call
	p := NewDisplay()
	wg := sync.WaitGroup{}

	total := int64(1024 * 1024 * 100)
	per := 1024 * 1024
	bar1 := p.AddBar("test1", 0, "done")
	if bar1 == nil {
		t.Fatal("create faield bar 1")
	}
	bar2 := p.AddBar("test2", total, "done")
	if bar2 == nil {
		t.Fatal("create faield bar 2")
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		offset := total
		for {
			<-time.After(time.Millisecond * 100)
			InCr(bar2, per, time.Millisecond*100)
			InCr(bar1, per, time.Millisecond*100)
			if offset <= 0 {

				break
			}
			offset -= int64(per)
		}
	}()

	wg.Wait()
	<-time.After(time.Second)
}
