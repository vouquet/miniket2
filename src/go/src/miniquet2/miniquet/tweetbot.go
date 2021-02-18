package miniquet

//import "log"

import (
	"io"
	"io/ioutil"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"time"
	"crypto/rand"
	"crypto/sha1"
	"crypto/hmac"
	"strings"
	"strconv"
	"encoding/base64"
)

import (
	"github.com/BurntSushi/toml"
)

type TweetBot struct {
	OauthConsumerKey     string
	OauthConsumerSecret  string
	OauthToken           string
	OauthAccessSecret    string
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
	return &TweetBot{
		OauthConsumerKey: c.ConsumerKey,
		OauthConsumerSecret: c.ConsumerSecret,
		OauthToken: c.Token,
		OauthAccessSecret: c.AccessSecret,
	}, nil
}

func (self *TweetBot) Tweet(msg string) error {
	params := map[string]string{"status": msg}
	header := self.generateHeader(params, "POST", "https://api.twitter.com/1.1/statuses/update.json")

	req, err := http.NewRequest("POST", "https://api.twitter.com/1.1/statuses/update.json", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", header)
	req.URL.RawQuery = sortedQueryString(params)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

/*
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Println(string(b))
	*/

	return nil
}

func mapMerge(m1, m2 map[string]string) map[string]string {
	m := map[string]string{}

	for k, v := range m1 {
		m[k] = v
	}
	for k, v := range m2 {
		m[k] = v
	}
	return m
}

func sortedQueryString(m map[string]string) string {
	type sortedQuery struct {
		m    map[string]string
		keys []string
	}

	sq := &sortedQuery{
		m:    m,
		keys: make([]string, len(m)),
	}

	var i int
	for key := range m {
		sq.keys[i] = key
		i++
	}
	sort.Strings(sq.keys)

	values := make([]string, len(sq.keys))
	for i, key := range sq.keys {
		values[i] = fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(sq.m[key]))
	}
	return strings.Join(values, "&")
}

func (self *TweetBot) generateHeader(params map[string]string, method string, uri string) string {
	m := map[string]string{}
	m["oauth_consumer_key"] = self.OauthConsumerKey
	m["oauth_nonce"] = createoauthNonce()
	m["oauth_signature_method"] = "HMAC-SHA1"
	m["oauth_timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	m["oauth_token"] = self.OauthToken
	m["oauth_version"] = "1.0"

	baseQueryString := sortedQueryString(mapMerge(m, params))

	base := []string{}
	base = append(base, url.QueryEscape(method))
	base = append(base, url.QueryEscape(uri))
	base = append(base, url.QueryEscape(baseQueryString))

	signatureBase := strings.Join(base, "&")
	signatureKey := url.QueryEscape(self.OauthConsumerSecret) + "&" + url.QueryEscape(self.OauthAccessSecret)
	m["oauth_signature"] = calcHMACSHA1(signatureBase, signatureKey)

	header := fmt.Sprintf("OAuth oauth_consumer_key=\"%s\", oauth_nonce=\"%s\", oauth_signature=\"%s\", oauth_signature_method=\"%s\", oauth_timestamp=\"%s\", oauth_token=\"%s\", oauth_version=\"%s\"",
		url.QueryEscape(m["oauth_consumer_key"]),
		url.QueryEscape(m["oauth_nonce"]),
		url.QueryEscape(m["oauth_signature"]),
		url.QueryEscape(m["oauth_signature_method"]),
		url.QueryEscape(m["oauth_timestamp"]),
		url.QueryEscape(m["oauth_token"]),
		url.QueryEscape(m["oauth_version"]),
	)
	return header
}

func createoauthNonce() string {
	key := make([]byte, 32)
	rand.Read(key)

	a_nonce := base64.StdEncoding.EncodeToString(key)
	a_nonce = strings.Replace(a_nonce, "+", "", -1)
	a_nonce = strings.Replace(a_nonce, "/", "", -1)
	a_nonce = strings.Replace(a_nonce, "=", "", -1)
	return a_nonce
}
func calcHMACSHA1(base, key string) string {
	b := []byte(key)
	h := hmac.New(sha1.New, b)
	io.WriteString(h, base)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
	}
