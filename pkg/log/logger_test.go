package log

import "testing"

func Test_log(t *testing.T) {
	err := DevLog()
	if err != nil {
		t.Error(err)
	}
	SetLogLevel("info")
	Info("test")
	Debug("test")
	Warn("test")
	Error("test")
	SetLogLevel("debug")
	Debug("debug")
}
