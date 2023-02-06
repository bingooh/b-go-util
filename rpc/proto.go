package rpc

import (
	"github.com/bingooh/b-go-util/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func IsZero(m proto.Message) bool {
	return proto.Size(m) == 0
}

func GetMsgFullName(m proto.Message) string {
	return string(proto.MessageName(m))
}

func MustMarshal(m proto.Message) []byte {
	data, err := proto.Marshal(m)
	util.AssertNilErr(err, `protobuf序列化出错`)
	return data
}

func MustUnmarshal(data []byte, m proto.Message) {
	util.AssertNilErr(proto.Unmarshal(data, m), `protobuf反序列化出错`)
}

func MustNewAny(m proto.Message) *anypb.Any {
	any, err := anypb.New(m)
	util.AssertNilErr(err)
	return any
}

func MarshalToAny(m proto.Message) ([]byte, error) {
	any, err := anypb.New(m)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(any)
}

func UnmarshalAny(data []byte) (any *anypb.Any, err error) {
	any = new(anypb.Any)
	err = proto.Unmarshal(data, any)
	return
}

func UnmarshalAnyValue(data []byte) (proto.Message, error) {
	any, err := UnmarshalAny(data)
	if err != nil {
		return nil, err
	}

	return any.UnmarshalNew()
}
