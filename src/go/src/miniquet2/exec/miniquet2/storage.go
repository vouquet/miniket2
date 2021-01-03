package main

import (
	"io"
	"fmt"
	"path/filepath"
	"sync"
	"bytes"
)

import (
	"github.com/ugorji/go/codec"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	DefaultStorageOpt *StorageOpt = &StorageOpt{ErrorIfExist: false}
	MsgpckHndl *codec.MsgpackHandle = &codec.MsgpackHandle{}
)

func init() {
	MsgpckHndl.RawToString = true
}

type Storage struct {
	db  *leveldb.DB

	mtx *sync.Mutex
}

type StorageOpt struct {
	ErrorIfExist bool
}

func OpenStorage(path string, opt *StorageOpt) (*Storage, error) {
	if opt == nil {
		opt = DefaultStorageOpt
	}

	c_path := filepath.Clean(path)
	db, err := leveldb.OpenFile(c_path, nil)
	if err != nil {
		return nil, err
	}

	return &Storage{
		db: db,
		mtx: new(sync.Mutex),
	}, nil
}

func (self *Storage) Close() error {
	self.lock()
	defer self.unlock()

	if err := self.db.Close(); err != nil {
		return err
	}
	self.db = nil
	return nil
}

func (self *Storage) Put(entry *Entry) error {
	self.lock()
	defer self.unlock()

	if self.db == nil {
		return fmt.Errorf("target database is nil pointer.")
	}

	b, err := encode(entry)
	if err != nil {
		return err
	}

	id := []byte(entry.Id())
	return self.db.Put(id, b, nil)
}

func (self *Storage) Delete(entry *Entry) error {
	self.lock()
	defer self.unlock()

	if self.db == nil {
		return fmt.Errorf("target database is nil pointer.")
	}

	id := []byte(entry.Id())
	return self.db.Delete(id, nil)
}

func (self *Storage) Walk() ([]*Entry, error) {
	self.lock()
	defer self.unlock()

	if self.db == nil {
		return nil, fmt.Errorf("target database is nil pointer.")
	}

	iter := self.db.NewIterator(nil, nil)
	defer iter.Release()

	es := []*Entry{}
	for iter.Next() {
		v := iter.Value()

		e, err := decode(v)
		if err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return es, nil
}

func (self *Storage) Get(key string) (*Entry, error) {
	self.lock()
	defer self.unlock()

	if self.db == nil {
		return nil, fmt.Errorf("target database is nil pointer.")
	}

	val, err := self.db.Get([]byte(key), nil)
	if err != nil {
		return nil, err
	}

	return decode(val)
}

func (self *Storage) lock() {
	self.mtx.Lock()
}

func (self *Storage) unlock() {
	self.mtx.Unlock()
}

func encode(e *Entry) ([]byte, error) {
	buf := new(bytes.Buffer)
	var w io.Writer = buf

	if err := codec.NewEncoder(w, MsgpckHndl).Encode(e); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decode(b []byte) (*Entry, error) {
	var e Entry
	if err := codec.NewDecoderBytes(b, MsgpckHndl).Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}
