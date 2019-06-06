package gohelpers

import (
	"github.com/zhuomouren/gohelpers/gocrypto"
	"github.com/zhuomouren/gohelpers/gofile"
	"github.com/zhuomouren/gohelpers/gonet"
	"github.com/zhuomouren/gohelpers/gostring"
	"github.com/zhuomouren/gohelpers/govalue"
)

var File = gofile.FileHelper

var String = gostring.Helper

var Crypto = gocrypto.Helper

var Net = gonet.Helper

func Value(value string) *govalue.GoValue {
	return govalue.NewGoValue(value)
}
