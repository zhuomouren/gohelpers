package gohelpers

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// assert an object must be a struct pointer
func panicAssertStructPtr(val reflect.Value) error {
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
		return nil
	}

	return fmt.Errorf("%s must be a struct pointer", val.Type().Name())
}

// set values from one struct to other struct
// both need ptr struct
func SetFormValues(from interface{}, to interface{}, skips ...string) error {
	val := reflect.ValueOf(from)
	elm := reflect.Indirect(val)

	valTo := reflect.ValueOf(to)
	elmTo := reflect.Indirect(valTo)

	if err := panicAssertStructPtr(val); err != nil {
		return err
	}
	if err := panicAssertStructPtr(valTo); err != nil {
		return err
	}

outFor:
	for i := 0; i < elmTo.NumField(); i++ {
		toF := elmTo.Field(i)
		name := elmTo.Type().Field(i).Name

		// skip specify field
		for _, skip := range skips {
			if skip == name {
				continue outFor
			}
		}
		f := elm.FieldByName(name)
		if f.Kind() != reflect.Invalid {
			// set value if type matched
			if f.Type().String() == toF.Type().String() {
				toF.Set(f)
			} else {
				fInt := false
				switch f.Interface().(type) {
				case int, int8, int16, int32, int64:
					fInt = true
				case uint, uint8, uint16, uint32, uint64:
				default:
					continue outFor
				}
				switch toF.Interface().(type) {
				case int, int8, int16, int32, int64:
					var v int64
					if fInt {
						v = f.Int()
					} else {
						vu := f.Uint()
						if vu > math.MaxInt64 {
							continue outFor
						}
						v = int64(vu)
					}
					if toF.OverflowInt(v) {
						continue outFor
					}
					toF.SetInt(v)
				case uint, uint8, uint16, uint32, uint64:
					var v uint64
					if fInt {
						vu := f.Int()
						if vu < 0 {
							continue outFor
						}
						v = uint64(vu)
					} else {
						v = f.Uint()
					}
					if toF.OverflowUint(v) {
						continue outFor
					}
					toF.SetUint(v)
				}
			}
		}
	}

	return nil
}

// compare field values between two struct pointer
// return changed field names
func FormChanges(base interface{}, modified interface{}, skips ...string) (fields []string, err error) {
	val := reflect.ValueOf(base)
	elm := reflect.Indirect(val)

	valMod := reflect.ValueOf(modified)
	elmMod := reflect.Indirect(valMod)

	err = panicAssertStructPtr(val)
	if err != nil {
		return
	}
	err = panicAssertStructPtr(valMod)
	if err != nil {
		return
	}

outFor:
	for i := 0; i < elmMod.NumField(); i++ {
		modF := elmMod.Field(i)
		name := elmMod.Type().Field(i).Name

		fT := elmMod.Type().Field(i)

		for _, v := range strings.Split(fT.Tag.Get("form"), ";") {
			v = strings.TrimSpace(v)
			if v == "-" {
				continue outFor
			}
		}

		// skip specify field
		for _, skip := range skips {
			if skip == name {
				continue outFor
			}
		}
		f := elm.FieldByName(name)
		if f.Kind() == reflect.Invalid {
			continue
		}

		// compare two values use string
		if fmt.Sprintf("%v", modF.Interface()) != fmt.Sprintf("%v", f.Interface()) {
			fields = append(fields, name)
		}
	}

	return
}
