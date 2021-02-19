package miniquet

import (
	"sync"
	"path/filepath"
)

import (
	"github.com/BurntSushi/toml"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/go-twitter/twitter"
)

type TweetBot struct {
	cl  *twitter.Client
	mtx *sync.Mutex
}

type TwConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	Token          string
	AccessSecret   string
}

func loadtwConfig(path string) (*TwConfig, error) {
	fpath := filepath.Clean(path)

	var conf TwConfig
	if _, err := toml.DecodeFile(fpath, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func NewTweetBot() (*TweetBot, error) {
	path := filepath.Clean("./.tw_config")

	c, err := loadtwConfig(path)
	if err != nil {
		return nil, err
	}

	config := oauth1.NewConfig(c.ConsumerKey, c.ConsumerSecret)
	token := oauth1.NewToken(c.Token, c.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	client := twitter.NewClient(httpClient)

	return &TweetBot{
		cl: client,
		mtx: new(sync.Mutex),
	}, nil
}

func (self *TweetBot) Tweet(msg string) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	_, _, err := self.cl.Statuses.Update(msg, nil)
	return err
}
