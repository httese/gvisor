// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pkg/eventchannel/event.proto

package gvisor

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type DebugEvent struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Text                 string   `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DebugEvent) Reset()         { *m = DebugEvent{} }
func (m *DebugEvent) String() string { return proto.CompactTextString(m) }
func (*DebugEvent) ProtoMessage()    {}
func (*DebugEvent) Descriptor() ([]byte, []int) {
	return fileDescriptor_fcfbd51abd9de962, []int{0}
}

func (m *DebugEvent) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DebugEvent.Unmarshal(m, b)
}
func (m *DebugEvent) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DebugEvent.Marshal(b, m, deterministic)
}
func (m *DebugEvent) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DebugEvent.Merge(m, src)
}
func (m *DebugEvent) XXX_Size() int {
	return xxx_messageInfo_DebugEvent.Size(m)
}
func (m *DebugEvent) XXX_DiscardUnknown() {
	xxx_messageInfo_DebugEvent.DiscardUnknown(m)
}

var xxx_messageInfo_DebugEvent proto.InternalMessageInfo

func (m *DebugEvent) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DebugEvent) GetText() string {
	if m != nil {
		return m.Text
	}
	return ""
}

func init() {
	proto.RegisterType((*DebugEvent)(nil), "gvisor.DebugEvent")
}

func init() { proto.RegisterFile("pkg/eventchannel/event.proto", fileDescriptor_fcfbd51abd9de962) }

var fileDescriptor_fcfbd51abd9de962 = []byte{
	// 103 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x29, 0xc8, 0x4e, 0xd7,
	0x4f, 0x2d, 0x4b, 0xcd, 0x2b, 0x49, 0xce, 0x48, 0xcc, 0xcb, 0x4b, 0xcd, 0x81, 0x70, 0xf4, 0x0a,
	0x8a, 0xf2, 0x4b, 0xf2, 0x85, 0xd8, 0xd2, 0xcb, 0x32, 0x8b, 0xf3, 0x8b, 0x94, 0x4c, 0xb8, 0xb8,
	0x5c, 0x52, 0x93, 0x4a, 0xd3, 0x5d, 0x41, 0x72, 0x42, 0x42, 0x5c, 0x2c, 0x79, 0x89, 0xb9, 0xa9,
	0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x60, 0x36, 0x48, 0xac, 0x24, 0xb5, 0xa2, 0x44, 0x82,
	0x09, 0x22, 0x06, 0x62, 0x27, 0xb1, 0x81, 0x0d, 0x31, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x17,
	0xee, 0x7f, 0xef, 0x64, 0x00, 0x00, 0x00,
}