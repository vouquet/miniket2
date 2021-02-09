package main

import (
	"os"
	"os/signal"
	"fmt"
	"sync"
	"context"
)

import (
	"github.com/nsf/termbox-go"
)

const (
	SIZE_INPUT_BUFFER int = 5
)

type Controller struct {
	pollevt  func()termbox.Event
	msg_buf  chan *Message
	msg_send chan *Message

	sig_ch   chan os.Signal
	sig_hdlr func()

	resize_hdlr func()

	prnt_err func(string, ...interface{})

	ctx     context.Context
	cancel  context.CancelFunc
	mtx     *sync.Mutex
}

func NewController(p_ctx context.Context, pollevt func() termbox.Event,
		msg_send chan *Message, prnt_err func(string, ...interface{})) (*Controller, error) {
	ctx, cancel := context.WithCancel(p_ctx)

	if pollevt == nil {
		return nil, fmt.Errorf("cannnot set nil address to poll event handler")
	}

	return &Controller{
		pollevt: pollevt,
		msg_buf: make(chan *Message, SIZE_INPUT_BUFFER),
		msg_send: msg_send,
		sig_ch: make(chan os.Signal),
		sig_hdlr: nil,
		prnt_err: prnt_err,
		ctx:ctx,
		cancel:cancel,
		mtx:new(sync.Mutex),
	}, nil
}

func (self *Controller) Run() {
	go self.run_sender()
	go self.run_recver()
}

func (self *Controller) Close() {
	self.lock()
	defer self.unlock()

	self.close()
	close(self.msg_buf)
}

func (self *Controller) callPrintErr(s string, msg ...interface{}) {
	go func() {
		msg := fmt.Sprintf(s, msg...)
		self.prnt_err(msg)
	}()
}

func (self *Controller) ResizeHandler(f func()) {
	self.lock()
	defer self.unlock()

	self.resize_hdlr = f
}

func (self *Controller) callResizeHandler() {
	self.lock()
	go func() {
		defer self.unlock()

		if self.resize_hdlr == nil {
			return
		}
		self.resize_hdlr()
	}()
}

func (self *Controller) SignalInterruptHandler(f func()) {
	self.lock()
	defer self.unlock()

	self.sig_hdlr = f
	signal.Notify(self.sig_ch, os.Interrupt)
}

func (self *Controller) callSignalHandler() {
	self.lock()
	go func() {
		defer self.unlock()

		if self.sig_hdlr == nil {
			return
		}
		self.sig_hdlr()
	}()
}

func (self *Controller) run_sender() {
	for {
		select {
		case <- self.ctx.Done():
			return
		case msg, ok := <- self.msg_buf:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}

			select {
			case <- self.ctx.Done():
				return
			case self.msg_send <- msg:
			}
		}
	}
}

func (self *Controller) run_recver() {
	ev_ch := make(chan termbox.Event)
	go func() {
		for {
			ev := self.pollevt()

			select {
			case <- self.ctx.Done():
				return
			case ev_ch <- ev:
			}
		}
	}()

	for {
		select {
		case <- self.ctx.Done():
			self.close()
			return

		case <- self.sig_ch:
			self.callSignalHandler()

		case ev := <- ev_ch:
			switch ev.Type {
			case termbox.EventError:
				self.callPrintErr("Controller.run_recver: %s", ev.Err)
			case termbox.EventResize:
				self.callResizeHandler()
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyCtrlC:
					self.callSignalHandler()
				default:
					msg := &Message{Key:ev.Key, Ch:ev.Ch}

					select {
					case <- self.ctx.Done():
					case self.msg_buf <- msg:
					}
				}
			}
		}
	}
}

func (self *Controller) close() {
	self.cancel()
}

func (self *Controller) lock() {
	self.mtx.Lock()
}

func (self *Controller) unlock() {
	self.mtx.Unlock()
}
