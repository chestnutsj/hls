package download

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/chestnutsj/hls/pkg/log"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
)

func textGenerator(length int) string {
	// rand.Seed(time.Now().UnixNano()) // 初始化随机数生成器

	// 可选的字符集，这里使用小写字母和数字
	charSet := "abcdefghijklmnopqrstuvwxyz0123456789"

	// 创建一个切片用于存储生成的字符
	var b strings.Builder
	for i := 0; i < length; i++ {
		// 生成一个随机索引
		index := rand.Intn(len(charSet))
		// 将随机索引对应的字符添加到切片中
		b.WriteByte(charSet[index])
	}

	// 将切片转换为字符串并返回
	return b.String()
}

func Test_Chunk(t *testing.T) {
	err := log.DevLog()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove("test.txt")
	}()
	_ = os.Remove("test.txt")
	chunk, err := NewChunk("test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer chunk.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		chunk.Run()

	}()

	dataLen := 1000
	data := []byte(textGenerator(dataLen))
	per := 100
	pos := 0

	for pos < dataLen {
		chunk.writeChan <- NewFileData(int64(pos), data[pos:pos+per], 0)
		pos += per
	}

	if pos != len(data) {
		pos -= per
		chunk.writeChan <- NewFileData(int64(pos), data[pos:], 0)
	}

	chunk.Exit()

	wg.Wait()

	// check
	file, err := os.Open("test.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 创建一个缓冲读取器
	reader := bufio.NewReader(file)

	// 读取整个文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	if !bytes.Equal(content, data) {
		t.Fatal("data not the same")
	}
}
