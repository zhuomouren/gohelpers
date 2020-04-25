package gohelpers

import (
	"github.com/zhuomouren/gohelpers/gocrypto"
	"github.com/zhuomouren/gohelpers/gofile"
	"github.com/zhuomouren/gohelpers/goform"
	"github.com/zhuomouren/gohelpers/gonet"
	"github.com/zhuomouren/gohelpers/gostring"
	"github.com/zhuomouren/gohelpers/govalue"
)

var File = gofile.Helper

var String = gostring.Helper

var Crypto = gocrypto.Helper

var Net = gonet.NetHelper
var URL = gonet.URLHelper
var HTTP = gonet.HTTPRequestHelper

var Form = goform.New()

func Value(value interface{}) *govalue.Value {
	return govalue.New(value)
}

func NewHTTP() *gonet.Request {
	return gonet.NewRequest()
}
