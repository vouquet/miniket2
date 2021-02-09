package main

import (
	"os"
	"os/user"
	"fmt"
	"flag"
	"path/filepath"
	"context"
)

import (
	"github.com/vouquet/brain"
	"github.com/vouquet/go-gmo-coin/gomocoin"
)

import (
	"miniquet2/miniquet"
)

const (
	MiniketName string = "miniquet2 v0.0.1"
)

var (
	Conf         *miniquet.Config
)

func miniquet2() error {
	c, cancel := context.WithCancel(context.Background())

	gmocoin, err := gomocoin.NewGoMOcoin(Conf.ApiKey, Conf.SecretKey, c)
	if err != nil {
		return err
	}

	daniel, err := brain.NewDaniel(gmocoin, c, cancel)
	if err != nil {
		return err
	}
	defer daniel.Close()

	return daniel.Run()
}

func die(s string, msg ...interface{}) {
	fmt.Fprintf(os.Stderr, s + "\n" , msg...)
	os.Exit(1)
}

func init() {
	var c_path string
	flag.StringVar(&c_path, "c", "", "config path.")
	flag.Parse()

	if flag.NArg() < 0 {
		die("usage : miniquet2 -c <config path>")
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
}

func main() {
	if err := miniquet2(); err != nil {
		die("%s", err)
	}
}
