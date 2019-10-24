package govalue

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Value struct {
	data string
}

func New(value interface{}) *Value {
	return &Value{
		data: toString(value),
	}
}

func (this *Value) Set(value interface{}) *Value {
	this.data = toString(value)
	return this
}

func (this *Value) String() string {
	return this.data
}

func (this *Value) Bytes() []byte {
	return []byte(this.data)
}

func (this *Value) StrictBool() (bool, error) {
	str := strings.ToLower(this.data)
	if str == "on" {
		return true, nil
	}

	return strconv.ParseBool(str)
}
func (this *Value) Bool() bool {
	val, _ := this.StrictBool()
	return val
}

func (this *Value) StrictFloat32() (float32, error) {
	val, err := strconv.ParseFloat(this.String(), 32)
	return float32(val), err
}
func (this *Value) Float32() float32 {
	val, _ := this.StrictFloat32()
	return val
}

func (this *Value) StrictFloat64() (float64, error) {
	return strconv.ParseFloat(this.String(), 64)
}
func (this *Value) Float64() float64 {
	val, _ := this.StrictFloat64()
	return val
}

func (this *Value) StrictInt() (int, error) {
	val, err := strconv.ParseInt(this.String(), 10, 32)
	return int(val), err
}
func (this *Value) Int() int {
	val, _ := this.StrictInt()
	return val
}

func (this *Value) StrictInt8() (int8, error) {
	val, err := strconv.ParseInt(this.String(), 10, 8)
	return int8(val), err
}
func (this *Value) Int8() int8 {
	val, _ := this.StrictInt8()
	return val
}

func (this *Value) StrictInt16() (int16, error) {
	val, err := strconv.ParseInt(this.String(), 10, 16)
	return int16(val), err
}
func (this *Value) Int16() int16 {
	val, _ := this.StrictInt16()
	return val
}

func (this *Value) StrictInt32() (int32, error) {
	val, err := strconv.ParseInt(this.String(), 10, 32)
	return int32(val), err
}
func (this *Value) Int32() int32 {
	val, _ := this.StrictInt32()
	return val
}

func (this *Value) StrictInt64() (int64, error) {
	val, err := strconv.ParseInt(this.String(), 10, 64)
	return int64(val), err
}
func (this *Value) Int64() int64 {
	val, _ := this.StrictInt64()
	return val
}

func (this *Value) StrictUint() (uint, error) {
	val, err := strconv.ParseUint(this.String(), 10, 32)
	return uint(val), err
}
func (this *Value) Uint() uint {
	val, _ := this.StrictUint()
	return val
}

func (this *Value) StrictUint8() (uint8, error) {
	val, err := strconv.ParseUint(this.String(), 10, 8)
	return uint8(val), err
}
func (this *Value) Uint8() uint8 {
	val, _ := this.StrictUint8()
	return val
}

func (this *Value) StrictUint16() (uint16, error) {
	val, err := strconv.ParseUint(this.String(), 10, 16)
	return uint16(val), err
}
func (this *Value) Uint16() uint16 {
	val, _ := this.StrictUint16()
	return val
}

func (this *Value) StrictUint32() (uint32, error) {
	val, err := strconv.ParseUint(this.String(), 10, 32)
	return uint32(val), err
}
func (this *Value) Uint32() uint32 {
	val, _ := this.StrictUint32()
	return val
}

func (this *Value) StrictUint64() (uint64, error) {
	val, err := strconv.ParseUint(this.String(), 10, 64)
	return uint64(val), err
}
func (this *Value) Uint64() uint64 {
	val, _ := this.StrictUint64()
	return val
}

// Duration returns time.Duration type value.
func (this *Value) StrictDuration() (time.Duration, error) {
	return time.ParseDuration(this.String())
}
func (this *Value) Duration() time.Duration {
	val, _ := this.StrictDuration()
	return val
}

// TimeFormat parses with given format and returns time.Time type value.
func (this *Value) StrictTimeFormat(format string) (time.Time, error) {
	return time.Parse(format, this.String())
}
func (this *Value) TimeFormat(format string) time.Time {
	val, _ := this.StrictTimeFormat(format)
	return val
}

// Time parses with RFC3339 format and returns time.Time type value.
func (this *Value) StrictTime() (time.Time, error) {
	return this.StrictTimeFormat(time.RFC3339)
}
func (this *Value) Time() time.Time {
	val, _ := this.StrictTime()
	return val
}

func toString(value interface{}) string {
	var s string
	switch v := value.(type) {
	case bool:
		s = strconv.FormatBool(v)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		s = strconv.FormatInt(int64(v), 10)
	case int8:
		s = strconv.FormatInt(int64(v), 10)
	case int16:
		s = strconv.FormatInt(int64(v), 10)
	case int32:
		s = strconv.FormatInt(int64(v), 10)
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint:
		s = strconv.FormatUint(uint64(v), 10)
	case uint8:
		s = strconv.FormatUint(uint64(v), 10)
	case uint16:
		s = strconv.FormatUint(uint64(v), 10)
	case uint32:
		s = strconv.FormatUint(uint64(v), 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case string:
		s = v
	case []byte:
		s = string(v)
	case time.Time:
		s = v.String()
	default:
		s = fmt.Sprintf("%v", v)
	}

	return s
}
