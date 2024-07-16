package m3u

import (
	"context"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/task"
	"net/url"
	"testing"
)

func Test_m3u(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	m3uUrl := "https://gobob-yasl.mushroomtrack.com/hls/tBJwPFpgDjiUSG1Rl0pNow/1722171140/34000/34872/34872.m3u8"
	urlStr, err := url.Parse(m3uUrl)
	if err != nil {
		t.Fatal(err)
	}
	m := NewM3uTask(ctx, nil, task.NewDownloadConfig(), urlStr, "./download")
	err = m.Start()
	if err != nil {
		t.Fatal(err)
	}
}
