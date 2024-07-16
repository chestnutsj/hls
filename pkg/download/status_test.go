package download

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/chestnutsj/hls/pkg/display"
	"github.com/chestnutsj/hls/pkg/log"
	"github.com/chestnutsj/hls/pkg/store"
	"github.com/chestnutsj/hls/pkg/tools"
	"os"
	"testing"
)

type jobStatus struct {
	offset int64
	l      int64
}

func (j *jobStatus) GetPos() int64 {
	return j.offset
}
func (j *jobStatus) GetData() []byte {
	return nil
}
func (j *jobStatus) GetDataLen() int {
	return 0
}
func (j *jobStatus) GetStart() int64 {
	return j.offset
}
func (j *jobStatus) GetOffsetLen() int64 {
	return j.l
}

func Test_Process(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}

	testData := []jobStatus{
		{

			offset: 10,
			l:      100,
		},
		{

			offset: 0,
			l:      10,
		},
		{

			offset: 300,
			l:      100,
		},
		{

			offset: 200,
			l:      100,
		},
	}
	total := int64(0)

	for _, x := range testData {
		total += x.l
	}

	show := display.NewDisplay()
	bar := show.AddBar("test", total, "down")
	if bar == nil {
		t.Fatal("bar is nil")
	}

	p := NewProgress(bar)

	err = p.InitCache("test", statusSuffix, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = p.cache.Close()
		_ = os.Remove("test.xz3")
	}()

	defer p.Close()

	for _, data := range testData {
		p.UpdateStatus(&data)
	}

	for _, data := range testData {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(data.offset))
		if v, err := p.cache.Get(string(key)); err != nil {
			t.Fatal(err, data.offset, "key ", string(key))
		} else {
			l := int64(0)
			if len(v) != 0 {
				l = int64(binary.BigEndian.Uint64(v))
			}
			if l != data.l {
				t.Errorf("can't get %d %d", l, data.l)
			}
		}
	}

	p.cache.Fetch(func(key string, value []byte) bool {
		if key == "status" {
			return false
		}

		if len(value) > 0 {
			t.Logf("{%d} {%d}", binary.BigEndian.Uint64([]byte(key)), binary.BigEndian.Uint32(value))
		} else {
			t.Logf("{%d} {%d}", binary.BigEndian.Uint64([]byte(key)), 0)
		}

		return false
	})

}

func Test_cache(t *testing.T) {
	t.Skip("not check")
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}

	prof := NewProgress(nil)
	meta := "{\"Url\":{\"Scheme\":\"https\",\"Opaque\":\"\",\"User\":null,\"Host\":\"ak.hycdn.cn\",\"Path\":\"/apk/202405291644-2281-ph9okkgrrl7cqazll4mi/arknights-hg-2281.apk\",\"RawPath\":\"\",\"OmitHost\":false,\"ForceQuery\":false,\"RawQuery\":\"\",\"Fragment\":\"\",\"RawFragment\":\"\"},\"FileName\":\"arknights-hg-2281.apk\",\"SourceFile\":\"arknights-hg-2281.apk\"}"
	err = prof.InitCache("test", statusSuffix, []byte(meta))
	if err != nil {
		t.Fatal(err)
	}
	l := prof.cache.Len()
	t.Log("cache len", l)
	total := int64(1895613825)
	chunkSize := int64(1024 * 1024 * 10)
	m := make(map[int64]int64)
	tools.AddUncovered(m, 0, total, chunkSize)

	t.Log(len(m))
	comped := make([]int64, 0)
	countFound := 0
	for k, v := range m {
		var key bytes.Buffer
		_ = binary.Write(&key, binary.BigEndian, k)

		val, err := prof.cache.Get(key.String())
		if err != nil {
			if errors.Is(err, store.NotFound) {
				continue
			} else {
				t.Fatal(err)
			}
		}
		countFound += 1
		size := binary.BigEndian.Uint64(val)
		if (k + int64(size)) == v {
			comped = append(comped, k)
			t.Log(k, size)
			if (k + int64(size)) >= total {
				t.Log("over")
			}
		}
	}
	t.Log("has su", len(comped), countFound)
	if len(comped) == 0 {
		t.Fatal("not ")
	}
	res := prof.GetTasks(total, chunkSize)

	if len(res) > len(m) {
		t.Fatal("not get failed")
	}
}
