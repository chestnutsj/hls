package store

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

/*
*
memory

	map[key] = ValueMeta {
	            Pos, <-fileoffset
				valueLen,
	           }

file:

	    uint32 uint32  keyVale, ValueData
		keyLen,ValueLength,     value
*/

var NotFound = errors.New("not Found Key")

type ValueMeta struct {
	Pos int64
	Len uint32
}

type KeyVal = map[string]ValueMeta

type BitCask struct {
	log  *Log
	path string
	kV   KeyVal
	mu   sync.RWMutex
}

type Log struct {
	file *os.File
	mu   sync.RWMutex
	sync.Once
}

func NewBitCask(path string) (*BitCask, error) {

	log, err := newLog(path)
	if err != nil {
		return nil, err
	}
	kv, err := log.initCache()
	if err != nil {
		return nil, err
	}
	return &BitCask{path: path, log: log, kV: kv}, nil
}
func (bc *BitCask) GetPath() string {
	return bc.path
}

func (bc *BitCask) Len() int {
	bc.mu.RLock()
	bc.mu.RUnlock()
	return len(bc.kV)
}

func (bc *BitCask) Reset() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.log.truncate()
	kv, err := bc.log.initCache()
	if err != nil {
		return err
	}
	bc.kV = kv
	return nil
}

func (bc *BitCask) Close() error {
	return bc.log.Close()
}

func (bc *BitCask) Keys() [][]byte {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	keys := make([][]byte, 0, len(bc.kV))
	for k := range bc.kV {
		keys = append(keys, []byte(k))
	}
	return keys
}

func (bc *BitCask) Get(key string) ([]byte, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if meta, ok := bc.kV[key]; ok {
		return bc.log.readValue(meta.Pos, meta.Len)
	}
	return nil, NotFound
}

func (bc *BitCask) Fetch(f func(key string, value []byte) bool) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for k, v := range bc.kV {
		data, err := bc.log.readValue(v.Pos, v.Len)
		if err != nil {
			return err
		}
		if f(k, data) {
			break
		}
	}
	return nil
}

func (bc *BitCask) Set(key string, value []byte) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	pos, l, err := bc.log.append([]byte(key), value)
	if err != nil {
		return err
	}
	bc.kV[key] = ValueMeta{Pos: pos, Len: l}
	return nil
}

func (bc *BitCask) FetchBitMap(f func(key string, v ValueMeta) bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for k, v := range bc.kV {
		if f(k, v) {
			break
		}
	}
}

func newLog(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &Log{file: file}, nil
}

func (l *Log) truncate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.file.Truncate(0)
}

func (l *Log) Close() error {
	var err error
	l.Do(func() {
		err = l.file.Close()
	})
	return err
}

func (l *Log) initCache() (KeyVal, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	_, err := l.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	kv := make(KeyVal)

	for {
		keyLenBytes := make([]byte, 4)
		if _, err := io.ReadFull(l.file, keyLenBytes); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		keyLen := binary.BigEndian.Uint32(keyLenBytes)

		valueLenBytes := make([]byte, 4)
		if _, err := io.ReadFull(l.file, valueLenBytes); err != nil {
			return nil, err
		}
		valueLen := binary.BigEndian.Uint32(valueLenBytes)

		key := make([]byte, keyLen)
		if _, err := io.ReadFull(l.file, key); err != nil {
			return nil, err
		}
		pos, _ := l.file.Seek(0, io.SeekCurrent)
		kv[string(key)] = ValueMeta{Pos: pos, Len: valueLen}
		if valueLen > 0 {
			if _, err := l.file.Seek(int64(valueLen), io.SeekCurrent); err != nil {
				return nil, err
			}
		}
	}
	return kv, nil
}

func (l *Log) readValue(pos int64, len uint32) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len == 0 {
		return nil, nil
	}

	_, err := l.file.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, err
	}
	value := make([]byte, len)
	if _, err := io.ReadFull(l.file, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (l *Log) append(key []byte, value []byte) (int64, uint32, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, 0, err
	}

	if err := binary.Write(l.file, binary.BigEndian, uint32(len(key))); err != nil {
		return 0, 0, err
	}
	valueLen := uint32(len(value))

	if err := binary.Write(l.file, binary.BigEndian, valueLen); err != nil {
		return 0, 0, err
	}
	if _, err := l.file.Write(key); err != nil {
		return 0, 0, err
	}

	pos, _ := l.file.Seek(0, io.SeekCurrent)

	if valueLen > 0 {
		if _, err := l.file.Write(value); err != nil {
			return 0, 0, err
		}
	}
	return pos, valueLen, nil
}
