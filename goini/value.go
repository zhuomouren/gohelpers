package goini

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GetDefault(def ...interface{}) *Value {
	if len(def) > 0 {
		var s string
		switch v := def[0].(type) {
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
			// ini.logf(DEBUG, "default value[ %s ] unknown type", s)
		}

		return NewValue(s)
	}

	return NewValue("")
}

type Value struct {
	value string
}

func NewValue(value string) *Value {
	return &Value{
		value: value,
	}
}

func (v *Value) String() string {
	return v.value
}

func (v *Value) Bytes() []byte {
	return []byte(v.value)
}

func (v *Value) StrictBool() (bool, error) {
	str := strings.ToLower(v.value)
	if str == "on" {
		return true, nil
	}

	return strconv.ParseBool(str)
}
func (v *Value) Bool() bool {
	val, _ := v.StrictBool()
	return val
}

func (v *Value) StrictFloat32() (float32, error) {
	val, err := strconv.ParseFloat(v.String(), 32)
	return float32(val), err
}
func (v *Value) Float32() float32 {
	val, _ := v.StrictFloat32()
	return val
}

func (v *Value) StrictFloat64() (float64, error) {
	return strconv.ParseFloat(v.String(), 64)
}
func (v *Value) Float64() float64 {
	val, _ := v.StrictFloat64()
	return val
}

func (v *Value) StrictInt() (int, error) {
	val, err := strconv.ParseInt(v.String(), 10, 32)
	return int(val), err
}
func (v *Value) Int() int {
	val, _ := v.StrictInt()
	return val
}

func (v *Value) StrictInt8() (int8, error) {
	val, err := strconv.ParseInt(v.String(), 10, 8)
	return int8(val), err
}
func (v *Value) Int8() int8 {
	val, _ := v.StrictInt8()
	return val
}

func (v *Value) StrictInt16() (int16, error) {
	val, err := strconv.ParseInt(v.String(), 10, 16)
	return int16(val), err
}
func (v *Value) Int16() int16 {
	val, _ := v.StrictInt16()
	return val
}

func (v *Value) StrictInt32() (int32, error) {
	val, err := strconv.ParseInt(v.String(), 10, 32)
	return int32(val), err
}
func (v *Value) Int32() int32 {
	val, _ := v.StrictInt32()
	return val
}

func (v *Value) StrictInt64() (int64, error) {
	val, err := strconv.ParseInt(v.String(), 10, 64)
	return int64(val), err
}
func (v *Value) Int64() int64 {
	val, _ := v.StrictInt64()
	return val
}

func (v *Value) StrictUint() (uint, error) {
	val, err := strconv.ParseUint(v.String(), 10, 32)
	return uint(val), err
}
func (v *Value) Uint() uint {
	val, _ := v.StrictUint()
	return val
}

func (v *Value) StrictUint8() (uint8, error) {
	val, err := strconv.ParseUint(v.String(), 10, 8)
	return uint8(val), err
}
func (v *Value) Uint8() uint8 {
	val, _ := v.StrictUint8()
	return val
}

func (v *Value) StrictUint16() (uint16, error) {
	val, err := strconv.ParseUint(v.String(), 10, 16)
	return uint16(val), err
}
func (v *Value) Uint16() uint16 {
	val, _ := v.StrictUint16()
	return val
}

func (v *Value) StrictUint32() (uint32, error) {
	val, err := strconv.ParseUint(v.String(), 10, 32)
	return uint32(val), err
}
func (v *Value) Uint32() uint32 {
	val, _ := v.StrictUint32()
	return val
}

func (v *Value) StrictUint64() (uint64, error) {
	val, err := strconv.ParseUint(v.String(), 10, 64)
	return uint64(val), err
}
func (v *Value) Uint64() uint64 {
	val, _ := v.StrictUint64()
	return val
}

// Duration returns time.Duration type value.
func (v *Value) StrictDuration() (time.Duration, error) {
	return time.ParseDuration(v.String())
}
func (v *Value) Duration() time.Duration {
	val, _ := v.StrictDuration()
	return val
}

// TimeFormat parses with given format and returns time.Time type value.
func (v *Value) StrictTimeFormat(format string) (time.Time, error) {
	return time.Parse(format, v.String())
}
func (v *Value) TimeFormat(format string) time.Time {
	val, _ := v.StrictTimeFormat(format)
	return val
}

// Time parses with RFC3339 format and returns time.Time type value.
func (v *Value) StrictTime() (time.Time, error) {
	return v.StrictTimeFormat(time.RFC3339)
}
func (v *Value) Time() time.Time {
	val, _ := v.StrictTime()
	return val
}
