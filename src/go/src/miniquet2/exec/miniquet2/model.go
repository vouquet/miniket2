package main

import (
	"fmt"
	"time"
	"context"
	"strings"
)

import (
	"github.com/nsf/termbox-go"
	"github.com/hinoshiba/go-gmo-coin/gomocoin"
)

type Message struct {
	Key  termbox.Key
	Ch   rune
}

type Model struct {
	view  *View
	ctlr  *Controller
	msg_ch chan *Message

	v_st  *StatusViewLayer
	m_st  *StatusModel

	v_pg  *ProgressViewLayer
	m_pg  *ProgressModel

	v_log *LogViewLayer
	m_log *LogModel

	com_buf         string
	com_hdlr_add    func([]string)error
	com_hdlr_stop   func([]string)error
	com_hdlr_kill9  func([]string)error

	ctx    context.Context
	cancel context.CancelFunc
}

func NewModel(b_ctx context.Context) (*Model, error) {
	if b_ctx == nil {
		b_ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(b_ctx)

	v, err := NewView(termbox.Output256, termbox.ColorDefault, termbox.ColorDefault)
	if err != nil {
		return nil, fmt.Errorf("cannot create view interface: %s", err)
	}

	v_st := NewStatusViewLayer(1)
	v_pg := NewProgressViewLayer(2)
	v_log := NewLogViewLayer(1)
	m_st := NewStatusModel()
	m_pg := NewProgressModel()
	m_log := NewLogModel()

	v.SetTitle(MiniketName)
	v.AddViewLayer(v_st)
	m_st.ViewHandler(v_st.SetValues)
	v.AddViewLayer(v_pg)
	m_pg.ViewHandler(v_pg.SetValues)
	v.AddViewLayer(v_log)
	m_log.ViewHandler(v_log.SetValues)

	pollevt_f := v.GetFuncPollEvent()
	msg_ch := make(chan *Message)
	c, err := NewController(ctx, pollevt_f, msg_ch, m_log.WriteErrLog)
	if err != nil {
		return nil, fmt.Errorf("cannot create controller: %s", err)
	}

	self := &Model{
		view: v,
		ctlr: c,
		msg_ch: msg_ch,

		v_st: v_st,
		m_st: m_st,

		v_pg: v_pg,
		m_pg: m_pg,

		v_log: v_log,
		m_log: m_log,

		ctx: ctx,
		cancel: cancel,
	}

	self.ctlr.SignalInterruptHandler(cancel)
	self.ctlr.ResizeHandler(self.refresh)

	return self, nil
}

func (self *Model) ContextWithCancel() context.Context {
	ctx, _ := context.WithCancel(self.ctx)
	return ctx
}

func (self *Model) Run() {
	self.ctlr.Run()
	self.run_reflesher()
	self.run_keymanager()
}

func (self *Model) run_reflesher() {
	t := time.NewTicker(500 * time.Millisecond)
	go func() {
		defer t.Stop()

		for {
			select {
			case <- self.ctx.Done():
				return
			case <- t.C:
				self.refresh()
			}
		}
	}()
}

func (self *Model) run_keymanager() {
	com_ch := make(chan string)
	var command bool = false

	go self.run_operator(com_ch)

	for {
		select {
		case <- self.ctx.Done():
			return
		case msg, ok := <- self.msg_ch:
			if !ok {
				self.cancel()
				return
			}

			if !command {
				if string(msg.Ch) == ":" {
					self.com_buf = ""
					command = true
					self.view.SetOperand(":" + self.com_buf)
					continue
				}
			}

			switch msg.Key {
			case termbox.KeyBackspace2:
				if len(self.com_buf) < 1 {
					continue
				}

				last := len(self.com_buf) - 1

				if last < 1 {
					self.com_buf = ""
				} else {
					self.com_buf = self.com_buf[:last]
				}
			case termbox.KeyBackspace:
				if len(self.com_buf) < 1 {
					continue
				}

				last := len(self.com_buf) - 1
				if last < 1 {
					self.com_buf = ""
				} else {
					self.com_buf = self.com_buf[:last]
				}
			case termbox.KeyEnter:
				if len(self.com_buf) < 1 {
					continue
				}

				select {
				case <- self.ctx.Done():
				case com_ch <- self.com_buf:
				}

				self.view.SetOperand("")
				self.com_buf = ""
				command = false
				continue

			case termbox.KeyEsc:
				self.view.SetOperand("")
				self.com_buf = ""
				command = false
				continue

			case termbox.KeySpace:
				if !command {
					continue
				}
				self.com_buf += string(" ")
			default:
				if !command {
					continue
				}
				self.com_buf += string(msg.Ch)
			}

			if !command {
				continue
			}

			self.view.SetOperand(":" + self.com_buf)
		}
	}
}

func (self *Model) run_operator(com_ch chan string) {
	for {
		select {
		case <- self.ctx.Done():
			return
		case command, ok := <- com_ch:
			if !ok {
				return
			}

			c_s := strings.SplitN(command, " ", 5)
			switch c_s[0] {
			case "help":
				self.view.SetOperandMsg("show https://github.com/vouquet/miniquet2")
			case "add":
				if len(c_s) < 2 {
					self.WriteErrLog("add command error: not set parameter")
				}
				if err := self.run_commandHandlerAdd(c_s[1:]); err != nil {
					self.WriteErrLog("add command error: %s", err)
					continue
				}
			case "stop":
				if len(c_s) < 2 {
					self.WriteErrLog("stop command error: not set parameter")
				}
				if err := self.run_commandHandlerStop(c_s[1:]); err != nil {
					self.WriteErrLog("stop command error: %s", err)
					continue
				}
			case "kill9":
				if len(c_s) < 2 {
					self.WriteErrLog("kill9 command error: not set parameter")
				}
				if err := self.run_commandHandlerKill9(c_s[1:]); err != nil {
					self.WriteErrLog("kill9 command error: %s", err)
					continue
				}
			default:
				self.WriteErrLog("undefined operation: %s", command)
			}
		}
	}
}

func (self *Model) AddTrader(tr *Trader) error {
	return self.m_pg.Add(tr)
}

func (self *Model) RemoveTrader(tr *Trader) error {
	return self.m_pg.Remove(tr)
}

func (self *Model) CommandHandlerAdd(f func([]string)error) {
	self.com_hdlr_add = f
}

func (self *Model) CommandHandlerStop(f func([]string)error) {
	self.com_hdlr_stop = f
}

func (self *Model) CommandHandlerKill9(f func([]string)error) {
	self.com_hdlr_kill9 = f
}

func (self *Model) run_commandHandlerAdd(args []string) error {
	if self.com_hdlr_add == nil {
		return fmt.Errorf("run_commandHandlerAdd: function pointer is nil.")
	}

	if args == nil {
		return fmt.Errorf("run_commandHandlerAdd: does not have args.")
	}
	if len(args) < 1 {
		return fmt.Errorf("run_commandHandlerAdd: does not have args.")
	}

	res := self.com_hdlr_add(args)
	return res
}

func (self *Model) run_commandHandlerStop(args []string) error {
	if self.com_hdlr_stop == nil {
		return fmt.Errorf("run_commandHandlerStop: function pointer is nil.")
	}

	if args == nil {
		return fmt.Errorf("run_commandHandlerStop: does not have args.")
	}
	if len(args) < 1 {
		return fmt.Errorf("run_commandHandlerStop: does not have args.")
	}

	res := self.com_hdlr_stop(args)
	return res
}

func (self *Model) run_commandHandlerKill9(args []string) error {
	if self.com_hdlr_kill9 == nil {
		return fmt.Errorf("run_commandHandlerKill9: function pointer is nil.")
	}

	if args == nil {
		return fmt.Errorf("run_commandHandlerKill9: does not have args.")
	}
	if len(args) < 1 {
		return fmt.Errorf("run_commandHandlerKill9: does not have args.")
	}

	res := self.com_hdlr_kill9(args)
	return res
}

func (self *Model) refresh() {
	self.view.Resize()
	self.m_st.Publish()
	self.m_pg.Publish()
	self.m_log.Publish()

	self.view.SetTitle(MiniketName)
	if len(self.com_buf) < 1 {
		return
	}
	self.view.SetOperand(":" + self.com_buf)
}

func (self *Model) WriteErrLog(s string, msg ...interface{}) {
	self.m_log.WriteErrLog(s, msg...)
	self.view.SetOperandErr(s, msg...)
}

func (self *Model) WriteMsgLog(s string, msg ...interface{}) {
	self.m_log.WriteMsgLog(s, msg...)
}

func (self *Model) UpdateStatus(rates map[string]*gomocoin.RateData) {
	self.m_st.UpdateStatus(rates)
}

func (self *Model) Close() {
	self.cancel()
	self.ctlr.Close()
	self.view.Close()
}
