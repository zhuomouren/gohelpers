package gostore

import (
	"sync"

	"github.com/zhuomouren/gohelpers/govalue"
)

var Store = &store{}

type store struct {
	data sync.Map
}

func New() *store {
	return &store{}
}

func (this *store) Set(key, value interface{}) {
	this.data.Store(key, value)
}

func (this *store) Get(key interface{}, def ...interface{}) *govalue.Value {
	value, ok := this.data.Load(key)
	if ok {
		return govalue.New(value)
	}

	if len(def) > 0 {
		govalue.New(def[0])
	}

	return govalue.New("")
}
