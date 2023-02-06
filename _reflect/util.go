package _reflect

import (
	"errors"
	"github.com/bingooh/b-go-util/_interface"
	"reflect"
)

func IsNil(v interface{}) bool {
	if v == nil {
		return true
	}

	switch reflect.TypeOf(v).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(v).IsNil()
	}

	return false
}

// IsZero 是否为零值。如果比较零值struct，建议使用具体类型比较，如v==User{}
func IsZero(v interface{}) bool {
	if v == nil {
		return true
	}

	t := reflect.Indirect(reflect.ValueOf(v))
	return t.IsValid() && t.IsZero()
}

// IsPrimitive 是否为基本数据类型，指针不属于基本数据类型
func IsPrimitive(v interface{}) bool {
	if v == nil {
		return false
	}

	return IsPrimitiveKind(reflect.TypeOf(v).Kind())
}

// IsPrimitiveKind 是否为基本数据类型
func IsPrimitiveKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Array, reflect.Struct, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Ptr, reflect.UnsafePointer:
		return false
	default:
		return true
	}
}

// PluckStructFieldValue 获取struct字段值
// 参数record类型必须为struct/*struct，字段值仅支持基本数据类型或基本数据指针类型
// 使用参数fieldName匹配字段时，首先匹配对应的字段名称，如果找不到则匹配json标签值
func PluckStructFieldValue(record interface{}, fieldName string, removeZeroValue bool) (interface{}, bool) {
	if record == nil {
		return nil, false
	}

	v := reflect.Indirect(reflect.ValueOf(record))
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	fv := v.FieldByName(fieldName)
	if fv.Kind() == reflect.Ptr && fv.IsNil() {
		return nil, false
	}

	fv = reflect.Indirect(fv)
	if !fv.IsValid() {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Tag.Get(`json`) == fieldName {
				fv = reflect.Indirect(v.FieldByName(f.Name))
				break
			}
		}
	}

	if !fv.IsValid() ||
		!IsPrimitiveKind(fv.Kind()) ||
		fv.IsZero() && removeZeroValue {
		return nil, false
	}

	return fv.Interface(), true
}

// PluckStructFieldValues 获取struct字段值
func PluckStructFieldValues(fieldName string, removeZeroValue, removeDuplicateValue bool, records ...interface{}) (values []interface{}) {
	records = _interface.Flat(records...)
	if len(records) == 0 {
		return
	}

	m := make(map[interface{}]struct{})
	for _, record := range records {
		if v, ok := PluckStructFieldValue(record, fieldName, removeZeroValue); ok {
			if removeDuplicateValue {
				if _, exist := m[v]; exist {
					continue
				}

				m[v] = struct{}{}
			}

			values = append(values, v)
		}

	}

	return
}

func ConvertStructToMap(record interface{}, fieldNames ...string) map[string]interface{} {
	if record == nil || len(fieldNames) == 0 {
		return nil
	}

	m := make(map[string]interface{})
	for _, name := range fieldNames {
		if v, ok := PluckStructFieldValue(record, name, false); ok {
			m[name] = v
		}
	}

	return m
}

// ConvertStructListToMap 转换struct为map，参数result必须为非nil的map
func ConvertStructListToMap(keyFieldName, valFieldName string, result interface{}, records ...interface{}) (err error) {
	r := reflect.ValueOf(result)
	if r.Kind() != reflect.Map {
		return errors.New(`result is not map`)
	}

	if r.IsZero() {
		return errors.New(`result is nil`)
	}

	records = _interface.Flat(records...)
	if len(records) == 0 {
		return
	}

	for _, record := range records {
		key, ok := PluckStructFieldValue(record, keyFieldName, false)
		if !ok {
			continue
		}

		if len(valFieldName) == 0 {
			r.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(record))
			continue
		}

		if val, ok := PluckStructFieldValue(record, valFieldName, false); ok {
			r.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
		}
	}

	return
}
