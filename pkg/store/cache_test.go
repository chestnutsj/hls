package store

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func Test_cache(t *testing.T) {
	dir, err := os.MkdirTemp("", "bitcask")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	db := filepath.Join(dir, "test.db")
	bc, err := NewBitCask(db)
	if err != nil {
		t.Fatal(err)
	}

	testData := map[string]string{
		"test":  "value1",
		"test2": "value2",
		"test3": "",
		"test4": "value5",
	}

	for k, v := range testData {
		err = bc.Set(k, []byte(v))
		if err != nil {
			t.Fatal(err)
		}
	}

	for k, v := range testData {
		d, err := bc.Get(k)
		if err != nil {
			t.Fatal(err)
		}
		if string(d) != v {
			t.Fatalf("get failed %s %s", d, v)
		}
	}

	err = bc.Close()
	if err != nil {
		t.Fatal(err)
	}

	bc2, err := NewBitCask(db)
	if err != nil {
		t.Fatal(err)
	}

	defer bc2.Close()

	keys := bc2.Keys()
	if len(keys) != len(testData) {
		t.Fatalf("keys len %d != %d", len(keys), len(testData))
	}

	for _, k := range keys {
		v, err := bc2.Get(string(k))
		if err != nil {
			t.Fatal(err)
		}
		if string(v) != testData[string(k)] {
			t.Fatalf("key %s value %s != %s", k, v, testData[string(k)])
		} else {
			t.Logf("key {%s} value {%s} {%s} ", string(k), string(v), testData[string(k)])
		}
	}
}

func Test_decode(t *testing.T) {

	original := struct {
		F1 []byte
		F2 []byte
	}{
		F1: []byte("Hello"),
		F2: []byte("World"),
	}

	// 创建一个字节缓冲区来存储编码的数据
	var buf bytes.Buffer

	// 创建一个新的编码器，并将结构体编码到缓冲区中
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(original); err != nil {
		log.Fatal("Encoding Error:", err)
	}

	println(hex.Dump(buf.Bytes()))

}
