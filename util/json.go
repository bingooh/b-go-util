package util

import (
	stdJSON "encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/sjson"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func MustUnmarshalJSON(data []byte, v interface{}) {
	AssertNilErr(json.Unmarshal(data, v), `json反序列化出错`)
}

func MustMarshalJSON(v interface{}) []byte {
	data, err := MarshalJSON(v)
	AssertNilErr(err, `json序列化出错`)
	return data
}

func ToRawJSON(data interface{}) (stdJSON.RawMessage, bool) {
	switch v := data.(type) {
	case []byte:
		return v, true
	case stdJSON.RawMessage:
		return v, true
	case *stdJSON.RawMessage:
		return *v, true
	default:
		return nil, false
	}
}

func ToJsonReadable(data interface{}) interface{} {
	if v, ok := ToRawJSON(data); ok {
		return v
	}

	return data
}

func ToJsonBytes(v interface{}) ([]byte, error) {
	if data, ok := ToRawJSON(v); ok {
		return data, nil
	}

	return MarshalJSON(v)
}

//此错误可能来自GetJSONField().LastError()
func IsJSONFieldNotFoundErr(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), `not found`)
}

//获取JSON字段值,field支持路径，见jsonitor文档
func GetJSONField(data []byte, field string) jsoniter.Any {
	var fields []interface{}
	for _, v := range strings.Split(field, `.`) {
		fields = append(fields, v)
	}

	return json.Get(data, fields...)
}

//获取JSON字段值,field支持路径，见jsonitor文档，如果解析出错，返回可读性错误
//注：jsonitor转换整数字段值类似js，比如字段值为`11abc`，则调用q.ToInt()返回整数11，且q.LastError()为空
func ParseJSONField(tip string, data []byte, field string) (jsoniter.Any, error) {
	q := GetJSONField(data, field)

	//如果出错仍然返回q，兼容jsoniter
	if err := q.LastError(); err != nil {
		if IsJSONFieldNotFoundErr(err) {
			return q, NewNilError(`%v字段不存在[field=%v,data=%v]`, tip, field, string(data))
		}

		return q, NewIllegalArgError(err, `%v解析出错[field=%v,data=%v]`, tip, field, string(data))
	}

	return q, nil
}

//设置JSON字段值，field支持路径，见sjson库文档
func SetJSONField(data []byte, field string, val interface{}) (rs []byte, err error) {
	switch v := val.(type) {
	case []byte:
		rs, err = sjson.SetRawBytes(data, field, v)

		if err != nil {
			val = string(v) //便于显示错误日志
		}
	default:
		rs, err = sjson.SetBytes(data, field, v)
	}

	if err == nil {
		return rs, nil
	}

	return nil, fmt.Errorf(`设置JSON字段出错[field=%s,val=%v,data=%v]->%w`, field, val, string(data), err)
}

//删除JSON字段，field支持路径，见sjson库文档
func DelJSONField(data []byte, field string) (rs []byte, err error) {
	rs, err = sjson.DeleteBytes(data, field)
	if err != nil {
		err = fmt.Errorf(`删除JSON字段出错[field=%v,data=%v]`, field, string(data))
	}

	return
}
