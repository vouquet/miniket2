package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	LogTypeErr uint8 = 1
	LogTypeMsg uint8 = 0
	CacheSize  int = 10
	FmtTime string = "2006-01-02 15:04:05"
)

type LogModel struct {
	cache        []*LogValue

	cache_limit  int
	view_handler func([]*LogValue)

	mtx *sync.Mutex
}

func NewLogModel() *LogModel {
	return &LogModel{cache_limit:CacheSize, cache:make([]*LogValue, 0),
													mtx:new(sync.Mutex)}
}

func (self *LogModel) ViewHandler(f func([]*LogValue)) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.view_handler = f
}

func (self *LogModel) Publish() {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.publish()
}

func (self *LogModel) call_view_handler(lv []*LogValue) {
	if self.view_handler == nil {
		return
	}
	self.view_handler(lv)
}

func (self *LogModel) append(lv *LogValue) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if len(self.cache) >= self.cache_limit {
		sp := len(self.cache) - self.cache_limit
		self.cache = self.cache[sp:]
	}
	self.cache = append(self.cache, lv)

	self.publish()
}

func (self *LogModel) publish() {
	self.call_view_handler(self.cache)
}

func (self *LogModel) WriteErrLog(s string, msg ...interface{}) {
	lv := newErrLogValue(s, msg...)
	self.append(lv)
}

func (self *LogModel) WriteMsgLog(s string, msg ...interface{}) {
	lv := newMsgLogValue(s, msg...)
	self.append(lv)
}

type LogValue struct {
	t        time.Time
	log_type uint8
	log_msg  string
}

func newMsgLogValue(s string, msg ...interface{}) *LogValue {
	log_msg := fmt.Sprintf(s, msg...)
	return &LogValue{t:time.Now(), log_type:LogTypeMsg, log_msg:log_msg}
}

func newErrLogValue(s string, msg ...interface{}) *LogValue {
	log_msg := fmt.Sprintf(s, msg...)
	return &LogValue{t:time.Now(), log_type:LogTypeErr, log_msg:log_msg}
}

func (self *LogValue) Type() uint8 {
	return self.log_type
}

func (self *LogValue) Runes() []rune {
	msg := fmt.Sprintf("[%s] %s", self.t.Format(FmtTime), self.log_msg)
	return []rune(msg)
}
