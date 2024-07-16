package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/task"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chestnutsj/hls/pkg/log"
)

func sha256sum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()

	// 将文件内容流式复制到 hasher
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	// 计算哈希值
	hashBytes := hasher.Sum(nil)

	// 将哈希值转换为十六进制字符串
	return hex.EncodeToString(hashBytes), nil

}

func JobStart2(t *testing.T, stopAndStart bool) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}
	log.SetLogLevel("info")
	ctx := context.Background()

	defer func() {
		s, err := sha256sum("./arknights-hg-2281.apk")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(s)
		if s != "bfa153fd1cc0c5b2f401f044358d37eab002488c41777d2da5a7deec7644e319" {
			t.Fatal("check sha256 sum is different")
		}

		err = os.RemoveAll("./arknights-hg-2281.apk")
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll("./arknights-hg-2281.xz3")
		if err != nil {
			t.Fatal(err)
		}
	}()

	u, err := url.Parse("https://ak.hycdn.cn/apk/202405291644-2281-ph9okkgrrl7cqazll4mi/arknights-hg-2281.apk")
	if err != nil {
		t.Fatal(err)
	}
	// 使用strings.LastIndex或strings.Split来获取路径的最后一个元素
	lastIndex := strings.LastIndex(u.Path, "/")
	pageName := u.Path[lastIndex+1:]
	p := display.NewDisplay()
	errChan := make(chan error, 10)
	job1, cancel1 := context.WithCancel(ctx)
	wg := sync.WaitGroup{}
	wg.Add(1)
	newjob := func(jctx context.Context) {
		defer wg.Done()
		job := NewHttpTask(jctx, u, pageName, true, task.NewDownloadConfig(), p)
		if job == nil {
			errChan <- errors.New("create job ")

		}
		err = job.Start()
		if err != nil && err != context.Canceled {
			errChan <- err
		}
		t.Log("job exit")
	}
	go newjob(job1)
	if stopAndStart {
		<-time.After(time.Second * 10)
		cancel1()
		wg.Wait()
		t.Log("stop cancel1")
		<-time.After(time.Second * 10)
		job2, _ := context.WithCancel(ctx)
		wg.Add(1)
		go newjob(job2)

	}
	wg.Wait()

	t.Log("wait ")
	select {
	case err := <-errChan:
		{
			if err != nil {
				t.Fatal(err)
			}
		}
	default:
	}
}

func Test_Job_Start(t *testing.T) {
	JobStart2(t, false)
}

func Test_Job_Start2(t *testing.T) {
	JobStart2(t, true)
}
