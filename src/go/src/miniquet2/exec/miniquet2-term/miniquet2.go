package main

import "time"

import (
	"os"
	"os/user"
	"fmt"
	"flag"
	"path/filepath"
	"sync"
	"context"
	"strconv"
)

import (
	"github.com/vouquet/go-gmo-coin/gomocoin"
	"github.com/vouquet/brain"
)

import (
	"miniquet2/miniquet"
)

const (
	MiniketName string = "miniquet2-term v0.0.1"
)

var (
	StoragePath  string
	Conf         *miniquet.Config
)

type Miniket2 struct {
	m   *Model

	trs  map[string]*miniquet.Trader
	shop *gomocoin.GoMOcoin
	st   *miniquet.Storage
}

func NewMiniket2(api string, secret string, s_path string) (*Miniket2, error) {
	m, err := NewModel(context.Background())
	if err != nil {
		return nil, err
	}

	gmocoin, err := gomocoin.NewGoMOcoin(api, secret, m.ContextWithCancel())
	if err != nil {
		return nil, err
	}

	storage, err := miniquet.OpenStorage(s_path, nil)
	if err != nil {
		return nil, err
	}

	self := &Miniket2{
		m:m,
		trs: make(map[string]*miniquet.Trader),
		shop: gmocoin,
		st: storage,
	}

	if err := self.buildTrader(); err != nil {
		return nil, err
	}
	if err := self.buildCommand(); err != nil {
		return nil, err
	}
	if err := self.loadStorage(); err != nil {
		return nil, err
	}

	return self, nil
}

func (self *Miniket2) Run() {
	wg := new(sync.WaitGroup)

	self.run_model(wg)
	self.run_trader(wg)

	self.m.WriteMsgLog("started miniquet2")

	wg.Wait()
}

func (self *Miniket2) Close() error {
	self.m.Close()
	return self.st.Close()
}

func (self *Miniket2) run_model(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		self.m.Run()
	}()
}

func (self *Miniket2) run_trader(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx := self.m.ContextWithCancel()
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <- ctx.Done():
				return
			case <- t.C:
				go func() {
					rates, err := self.shop.UpdateRate()
					if err != nil {
						self.m.WriteErrLog("cannot update gmocoin: %s", err)
						return
					}

					self.m.UpdateStatus(rates)
					for _, t := range self.trs {
						go t.Do(self.m, rates)
					}
				}()
			}
		}
	}()
}

func (self *Miniket2) loadStorage() error {
	ens, err := self.st.Walk()
	if err != nil {
		return err
	}

	for _, en := range ens {
		tr, ok := self.trs[en.Trader]
		if !ok {
			return fmt.Errorf("cannt found '%s' trader.", en.Trader)
		}

		if err := tr.RequestAppend(en); err != nil {
			return fmt.Errorf("cannt append, %s", err)
		}
	}
	return nil
}

func (self *Miniket2) buildTrader() error {
	a_tr := miniquet.NewTrader("alice", "Trade with a difference of 0.2 point.", self.shop, self.st)
	a_tr.SetCheckFunc(brain.Alice)
	self.trs["alice"] = a_tr

	j_tr := miniquet.NewTrader("john", "Trade with a difference of 1 point.", self.shop, self.st)
	j_tr.SetCheckFunc(brain.John)
	self.trs["john"] = j_tr

	for _, tr := range self.trs {
		self.m.AddTrader(tr)
	}

	return nil
}

func (self *Miniket2) buildCommand() error {
	self.m.CommandHandlerAdd(func(args []string) error {
		if len(args) != 4 {
			return fmt.Errorf("args less than 4. USAGE: add <trader name> <symbol> <size> <buy rate>, %s, %s", len(args), args)
		}

		t_name := args[0]
		symbol := args[1]
		size, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		want_rate, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			return err
		}

		tr, ok := self.trs[t_name]
		if !ok {
			return fmt.Errorf("unkown trader name. :%s", t_name)
		}

		if err := tr.Add(symbol, size, want_rate); err != nil {
			return err
		}

		self.m.WriteMsgLog("added %s, %s, %v, %v", t_name, symbol, size, want_rate)
		return nil
	})

	self.m.CommandHandlerStop(func(args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("args less than 2. USAGE: stop <trader name> <id>")
		}

		t_name := args[0]
		id := args[1]
		tr, ok := self.trs[t_name]
		if !ok {
			return fmt.Errorf("unkown trader name. :%s", t_name)
		}

		if err := tr.RequestStop(id); err != nil {
			return err
		}

		self.m.WriteMsgLog("stoped : %s", id)
		return nil
	})

	self.m.CommandHandlerKill9(func(args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("args less than 2. USAGE: kill9 <trader name> <id>")
		}

		t_name := args[0]
		id := args[1]
		tr, ok := self.trs[t_name]
		if !ok {
			return fmt.Errorf("unkown trader name. :%s", t_name)
		}

		if err := tr.RequestKill9(id); err != nil {
			return err
		}

		self.m.WriteMsgLog("killed : %s", id)
		return nil
	})

	return nil
}

func die(s string, msg ...interface{}) {
	fmt.Fprintf(os.Stderr, s + "\n" , msg...)
	os.Exit(1)
}

func init() {
	var c_path string
	var r_path string
	flag.StringVar(&c_path, "c", "", "config path.")
	flag.StringVar(&r_path, "r", "./miniquet2.ldb", "record storage path.")
	flag.Parse()

	if flag.NArg() < 0 {
		die("usage : miniquet2 -c <config path> -b <record storage path>")
	}

	if r_path == "" {
		die("empty record storage path.")
	}
	if c_path == "" {
		usr, err := user.Current()
		if err != nil {
			die("cannot load string of path to user directory.")
		}
		c_path = usr.HomeDir + "/.miniquet2"
	}

	cfg, err := miniquet.LoadConfig(filepath.Clean(c_path))
	if err != nil {
		die("cannot load a config: %s", err)
	}

	Conf = cfg
	StoragePath  = r_path
}

func main() {
	m2, err := NewMiniket2(Conf.ApiKey, Conf.SecretKey, StoragePath)
	if err != nil {
		die("%s", err)
	}
	defer m2.Close()

	m2.Run()
}
