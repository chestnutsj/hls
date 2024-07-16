package download

import (
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileData interface {
	GetPos() int64
	GetData() []byte
	GetDataLen() int
	GetStart() int64
	GetOffsetLen() int64
}

type fileData struct {
	pos   int64
	data  []byte
	start int64
}

func (f *fileData) GetPos() int64 {
	return f.pos
}
func (f *fileData) GetData() []byte {
	return f.data
}
func (f *fileData) GetStart() int64 {
	return f.start
}
func (f *fileData) GetDataLen() int {
	return len(f.data)
}

func (f *fileData) GetOffsetLen() int64 {
	return (f.pos - f.start) + int64(len(f.data))
}

func NewFileData(pos int64, data []byte, start int64) FileData {
	val := make([]byte, len(data))
	copy(val, data)
	return &fileData{pos: pos, data: val, start: start}
}

type Chunk struct {
	path      string
	file      *os.File
	writeChan chan FileData
	sync.Mutex
	status *Progress
}

func NewChunk(path string, status *Progress) (*Chunk, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	writeChan := make(chan FileData, 100)
	return &Chunk{path: path, file: file, writeChan: writeChan, status: status}, nil
}

func (c *Chunk) Close() error {
	return c.file.Close()
}

func (c *Chunk) Exit() {
	close(c.writeChan)
}

func (c *Chunk) Run() {
	defer c.file.Sync()
	var err error
	for data := range c.writeChan {
		err = c.saveData(data)
		if err != nil {
			zap.L().Error("file error", zap.String("filename", c.path), zap.Error(err))
			return
		}
		if c.status != nil {
			c.status.UpdateStatus(data)
		}
	}
}

func (c *Chunk) saveData(data FileData) error {
	c.Lock()
	defer c.Unlock()
	_, err := c.file.Seek(data.GetPos(), io.SeekStart)
	if err != nil {

		return err
	}
	_, err = c.file.Write(data.GetData())
	if err != nil {
		return err
	}
	return nil
}
