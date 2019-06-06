package govalue

import (
	"strconv"
	"strings"
	"time"
)

type GoValue struct {
	value string
}

func NewGoValue(value string) *GoValue {
	return &GoValue{
		value: value,
	}
}

func (v *GoValue) String() string {
	return v.value
}

func (v *GoValue) Bytes() []byte {
	return []byte(v.value)
}

func (v *GoValue) StrictBool() (bool, error) {
	str := strings.ToLower(v.value)
	if str == "on" {
		return true, nil
	}

	return strconv.ParseBool(str)
}
func (v *GoValue) Bool() bool {
	val, _ := v.StrictBool()
	return val
}

func (v *GoValue) StrictFloat32() (float32, error) {
	val, err := strconv.ParseFloat(v.String(), 32)
	return float32(val), err
}
func (v *GoValue) Float32() float32 {
	val, _ := v.StrictFloat32()
	return val
}

func (v *GoValue) StrictFloat64() (float64, error) {
	return strconv.ParseFloat(v.String(), 64)
}
func (v *GoValue) Float64() float64 {
	val, _ := v.StrictFloat64()
	return val
}

func (v *GoValue) StrictInt() (int, error) {
	val, err := strconv.ParseInt(v.String(), 10, 32)
	return int(val), err
}
func (v *GoValue) Int() int {
	val, _ := v.StrictInt()
	return val
}

func (v *GoValue) StrictInt8() (int8, error) {
	val, err := strconv.ParseInt(v.String(), 10, 8)
	return int8(val), err
}
func (v *GoValue) Int8() int8 {
	val, _ := v.StrictInt8()
	return val
}

func (v *GoValue) StrictInt16() (int16, error) {
	val, err := strconv.ParseInt(v.String(), 10, 16)
	return int16(val), err
}
func (v *GoValue) Int16() int16 {
	val, _ := v.StrictInt16()
	return val
}

func (v *GoValue) StrictInt32() (int32, error) {
	val, err := strconv.ParseInt(v.String(), 10, 32)
	return int32(val), err
}
func (v *GoValue) Int32() int32 {
	val, _ := v.StrictInt32()
	return val
}

func (v *GoValue) StrictInt64() (int64, error) {
	val, err := strconv.ParseInt(v.String(), 10, 64)
	return int64(val), err
}
func (v *GoValue) Int64() int64 {
	val, _ := v.StrictInt64()
	return val
}

func (v *GoValue) StrictUint() (uint, error) {
	val, err := strconv.ParseUint(v.String(), 10, 32)
	return uint(val), err
}
func (v *GoValue) Uint() uint {
	val, _ := v.StrictUint()
	return val
}

func (v *GoValue) StrictUint8() (uint8, error) {
	val, err := strconv.ParseUint(v.String(), 10, 8)
	return uint8(val), err
}
func (v *GoValue) Uint8() uint8 {
	val, _ := v.StrictUint8()
	return val
}

func (v *GoValue) StrictUint16() (uint16, error) {
	val, err := strconv.ParseUint(v.String(), 10, 16)
	return uint16(val), err
}
func (v *GoValue) Uint16() uint16 {
	val, _ := v.StrictUint16()
	return val
}

func (v *GoValue) StrictUint32() (uint32, error) {
	val, err := strconv.ParseUint(v.String(), 10, 32)
	return uint32(val), err
}
func (v *GoValue) Uint32() uint32 {
	val, _ := v.StrictUint32()
	return val
}

func (v *GoValue) StrictUint64() (uint64, error) {
	val, err := strconv.ParseUint(v.String(), 10, 64)
	return uint64(val), err
}
func (v *GoValue) Uint64() uint64 {
	val, _ := v.StrictUint64()
	return val
}

// Duration returns time.Duration type value.
func (v *GoValue) StrictDuration() (time.Duration, error) {
	return time.ParseDuration(v.String())
}
func (v *GoValue) Duration() time.Duration {
	val, _ := v.StrictDuration()
	return val
}

// TimeFormat parses with given format and returns time.Time type value.
func (v *GoValue) StrictTimeFormat(format string) (time.Time, error) {
	return time.Parse(format, v.String())
}
func (v *GoValue) TimeFormat(format string) time.Time {
	val, _ := v.StrictTimeFormat(format)
	return val
}

// Time parses with RFC3339 format and returns time.Time type value.
func (v *GoValue) StrictTime() (time.Time, error) {
	return v.StrictTimeFormat(time.RFC3339)
}
func (v *GoValue) Time() time.Time {
	val, _ := v.StrictTime()
	return val
}
