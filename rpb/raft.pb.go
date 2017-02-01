// Code generated by protoc-gen-go.
// source: raft.proto
// DO NOT EDIT!

/*
Package rpb is a generated protocol buffer package.

It is generated from these files:
	raft.proto

It has these top-level messages:
	Nodes
	AppendRequest
	VoteRequest
	Entry
	Node
	Response
	Ack
*/
package rpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Nodes struct {
	SenderID int64   `protobuf:"varint,1,opt,name=senderID" json:"senderID,omitempty"`
	Nodes    []*Node `protobuf:"bytes,2,rep,name=nodes" json:"nodes,omitempty"`
}

func (m *Nodes) Reset()                    { *m = Nodes{} }
func (m *Nodes) String() string            { return proto.CompactTextString(m) }
func (*Nodes) ProtoMessage()               {}
func (*Nodes) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Nodes) GetSenderID() int64 {
	if m != nil {
		return m.SenderID
	}
	return 0
}

func (m *Nodes) GetNodes() []*Node {
	if m != nil {
		return m.Nodes
	}
	return nil
}

type AppendRequest struct {
	// sender ID
	SenderID int64 `protobuf:"varint,1,opt,name=senderID" json:"senderID,omitempty"`
	// leaders term
	Term int64 `protobuf:"varint,2,opt,name=term" json:"term,omitempty"`
	// leader ID
	LeaderID int64 `protobuf:"varint,3,opt,name=leaderID" json:"leaderID,omitempty"`
	// index of log entry immediately preceding new ones
	PrevLogIdx int64 `protobuf:"varint,4,opt,name=prevLogIdx" json:"prevLogIdx,omitempty"`
	// term of prevLogIdx entry
	PrevLogTerm int64 `protobuf:"varint,5,opt,name=prevLogTerm" json:"prevLogTerm,omitempty"`
	// leader commit index
	LeaderCommitIdx int64 `protobuf:"varint,6,opt,name=leaderCommitIdx" json:"leaderCommitIdx,omitempty"`
	LastEntryIdx    int64 `protobuf:"varint,7,opt,name=lastEntryIdx" json:"lastEntryIdx,omitempty"`
	// log entries to commit
	Entries []*Entry `protobuf:"bytes,8,rep,name=entries" json:"entries,omitempty"`
}

func (m *AppendRequest) Reset()                    { *m = AppendRequest{} }
func (m *AppendRequest) String() string            { return proto.CompactTextString(m) }
func (*AppendRequest) ProtoMessage()               {}
func (*AppendRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *AppendRequest) GetSenderID() int64 {
	if m != nil {
		return m.SenderID
	}
	return 0
}

func (m *AppendRequest) GetTerm() int64 {
	if m != nil {
		return m.Term
	}
	return 0
}

func (m *AppendRequest) GetLeaderID() int64 {
	if m != nil {
		return m.LeaderID
	}
	return 0
}

func (m *AppendRequest) GetPrevLogIdx() int64 {
	if m != nil {
		return m.PrevLogIdx
	}
	return 0
}

func (m *AppendRequest) GetPrevLogTerm() int64 {
	if m != nil {
		return m.PrevLogTerm
	}
	return 0
}

func (m *AppendRequest) GetLeaderCommitIdx() int64 {
	if m != nil {
		return m.LeaderCommitIdx
	}
	return 0
}

func (m *AppendRequest) GetLastEntryIdx() int64 {
	if m != nil {
		return m.LastEntryIdx
	}
	return 0
}

func (m *AppendRequest) GetEntries() []*Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

type VoteRequest struct {
	// candidate term
	Term int64 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	// sender's ID, the candidate
	CandidateID int64 `protobuf:"varint,2,opt,name=candidateID" json:"candidateID,omitempty"`
	// last log entry information
	LastLogIdx  int64 `protobuf:"varint,3,opt,name=lastLogIdx" json:"lastLogIdx,omitempty"`
	LastLogTerm int64 `protobuf:"varint,4,opt,name=lastLogTerm" json:"lastLogTerm,omitempty"`
}

func (m *VoteRequest) Reset()                    { *m = VoteRequest{} }
func (m *VoteRequest) String() string            { return proto.CompactTextString(m) }
func (*VoteRequest) ProtoMessage()               {}
func (*VoteRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *VoteRequest) GetTerm() int64 {
	if m != nil {
		return m.Term
	}
	return 0
}

func (m *VoteRequest) GetCandidateID() int64 {
	if m != nil {
		return m.CandidateID
	}
	return 0
}

func (m *VoteRequest) GetLastLogIdx() int64 {
	if m != nil {
		return m.LastLogIdx
	}
	return 0
}

func (m *VoteRequest) GetLastLogTerm() int64 {
	if m != nil {
		return m.LastLogTerm
	}
	return 0
}

type Entry struct {
	Term    int64  `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	Idx     int64  `protobuf:"varint,2,opt,name=idx" json:"idx,omitempty"`
	Command []byte `protobuf:"bytes,3,opt,name=command,proto3" json:"command,omitempty"`
}

func (m *Entry) Reset()                    { *m = Entry{} }
func (m *Entry) String() string            { return proto.CompactTextString(m) }
func (*Entry) ProtoMessage()               {}
func (*Entry) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *Entry) GetTerm() int64 {
	if m != nil {
		return m.Term
	}
	return 0
}

func (m *Entry) GetIdx() int64 {
	if m != nil {
		return m.Idx
	}
	return 0
}

func (m *Entry) GetCommand() []byte {
	if m != nil {
		return m.Command
	}
	return nil
}

type Node struct {
	ID   int64  `protobuf:"varint,1,opt,name=ID" json:"ID,omitempty"`
	Addr string `protobuf:"bytes,2,opt,name=addr" json:"addr,omitempty"`
}

func (m *Node) Reset()                    { *m = Node{} }
func (m *Node) String() string            { return proto.CompactTextString(m) }
func (*Node) ProtoMessage()               {}
func (*Node) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *Node) GetID() int64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *Node) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

type Response struct {
	SenderID       int64  `protobuf:"varint,1,opt,name=senderID" json:"senderID,omitempty"`
	Error          string `protobuf:"bytes,2,opt,name=error" json:"error,omitempty"`
	Accepted       bool   `protobuf:"varint,3,opt,name=accepted" json:"accepted,omitempty"`
	Term           int64  `protobuf:"varint,4,opt,name=term" json:"term,omitempty"`
	LeaderID       int64  `protobuf:"varint,5,opt,name=leaderID" json:"leaderID,omitempty"`
	LastAppliedIdx int64  `protobuf:"varint,6,opt,name=lastAppliedIdx" json:"lastAppliedIdx,omitempty"`
}

func (m *Response) Reset()                    { *m = Response{} }
func (m *Response) String() string            { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()               {}
func (*Response) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *Response) GetSenderID() int64 {
	if m != nil {
		return m.SenderID
	}
	return 0
}

func (m *Response) GetError() string {
	if m != nil {
		return m.Error
	}
	return ""
}

func (m *Response) GetAccepted() bool {
	if m != nil {
		return m.Accepted
	}
	return false
}

func (m *Response) GetTerm() int64 {
	if m != nil {
		return m.Term
	}
	return 0
}

func (m *Response) GetLeaderID() int64 {
	if m != nil {
		return m.LeaderID
	}
	return 0
}

func (m *Response) GetLastAppliedIdx() int64 {
	if m != nil {
		return m.LastAppliedIdx
	}
	return 0
}

type Ack struct {
	SenderID int64 `protobuf:"varint,1,opt,name=senderID" json:"senderID,omitempty"`
}

func (m *Ack) Reset()                    { *m = Ack{} }
func (m *Ack) String() string            { return proto.CompactTextString(m) }
func (*Ack) ProtoMessage()               {}
func (*Ack) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *Ack) GetSenderID() int64 {
	if m != nil {
		return m.SenderID
	}
	return 0
}

func init() {
	proto.RegisterType((*Nodes)(nil), "rpb.Nodes")
	proto.RegisterType((*AppendRequest)(nil), "rpb.AppendRequest")
	proto.RegisterType((*VoteRequest)(nil), "rpb.VoteRequest")
	proto.RegisterType((*Entry)(nil), "rpb.Entry")
	proto.RegisterType((*Node)(nil), "rpb.Node")
	proto.RegisterType((*Response)(nil), "rpb.Response")
	proto.RegisterType((*Ack)(nil), "rpb.Ack")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Raft service

type RaftClient interface {
	AppendEntries(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*Response, error)
	RequestVote(ctx context.Context, in *VoteRequest, opts ...grpc.CallOption) (*Response, error)
	Join(ctx context.Context, in *Node, opts ...grpc.CallOption) (*Nodes, error)
	ClusterState(ctx context.Context, in *Nodes, opts ...grpc.CallOption) (*Nodes, error)
	Leave(ctx context.Context, in *Node, opts ...grpc.CallOption) (*Ack, error)
}

type raftClient struct {
	cc *grpc.ClientConn
}

func NewRaftClient(cc *grpc.ClientConn) RaftClient {
	return &raftClient{cc}
}

func (c *raftClient) AppendEntries(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/rpb.Raft/AppendEntries", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *raftClient) RequestVote(ctx context.Context, in *VoteRequest, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/rpb.Raft/RequestVote", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *raftClient) Join(ctx context.Context, in *Node, opts ...grpc.CallOption) (*Nodes, error) {
	out := new(Nodes)
	err := grpc.Invoke(ctx, "/rpb.Raft/Join", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *raftClient) ClusterState(ctx context.Context, in *Nodes, opts ...grpc.CallOption) (*Nodes, error) {
	out := new(Nodes)
	err := grpc.Invoke(ctx, "/rpb.Raft/ClusterState", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *raftClient) Leave(ctx context.Context, in *Node, opts ...grpc.CallOption) (*Ack, error) {
	out := new(Ack)
	err := grpc.Invoke(ctx, "/rpb.Raft/Leave", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Raft service

type RaftServer interface {
	AppendEntries(context.Context, *AppendRequest) (*Response, error)
	RequestVote(context.Context, *VoteRequest) (*Response, error)
	Join(context.Context, *Node) (*Nodes, error)
	ClusterState(context.Context, *Nodes) (*Nodes, error)
	Leave(context.Context, *Node) (*Ack, error)
}

func RegisterRaftServer(s *grpc.Server, srv RaftServer) {
	s.RegisterService(&_Raft_serviceDesc, srv)
}

func _Raft_AppendEntries_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppendRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).AppendEntries(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rpb.Raft/AppendEntries",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).AppendEntries(ctx, req.(*AppendRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Raft_RequestVote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VoteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).RequestVote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rpb.Raft/RequestVote",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).RequestVote(ctx, req.(*VoteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Raft_Join_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Node)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).Join(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rpb.Raft/Join",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).Join(ctx, req.(*Node))
	}
	return interceptor(ctx, in, info, handler)
}

func _Raft_ClusterState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Nodes)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).ClusterState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rpb.Raft/ClusterState",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).ClusterState(ctx, req.(*Nodes))
	}
	return interceptor(ctx, in, info, handler)
}

func _Raft_Leave_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Node)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).Leave(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rpb.Raft/Leave",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).Leave(ctx, req.(*Node))
	}
	return interceptor(ctx, in, info, handler)
}

var _Raft_serviceDesc = grpc.ServiceDesc{
	ServiceName: "rpb.Raft",
	HandlerType: (*RaftServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AppendEntries",
			Handler:    _Raft_AppendEntries_Handler,
		},
		{
			MethodName: "RequestVote",
			Handler:    _Raft_RequestVote_Handler,
		},
		{
			MethodName: "Join",
			Handler:    _Raft_Join_Handler,
		},
		{
			MethodName: "ClusterState",
			Handler:    _Raft_ClusterState_Handler,
		},
		{
			MethodName: "Leave",
			Handler:    _Raft_Leave_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "raft.proto",
}

func init() { proto.RegisterFile("raft.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 489 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x54, 0x4b, 0x6e, 0xdb, 0x30,
	0x10, 0x85, 0x2c, 0x29, 0x56, 0x46, 0x4e, 0x1a, 0x10, 0x5d, 0x08, 0x46, 0x3f, 0xae, 0x50, 0x04,
	0x46, 0x17, 0x5e, 0xb8, 0x27, 0x30, 0xe2, 0xa0, 0x70, 0x11, 0x74, 0xa1, 0x16, 0xdd, 0xd3, 0xe2,
	0xa4, 0x10, 0x62, 0x89, 0x2c, 0xc5, 0x04, 0xe9, 0x01, 0x7a, 0x80, 0xde, 0xa4, 0x37, 0xe9, 0x95,
	0x0a, 0x0e, 0x25, 0x85, 0x36, 0x5a, 0xef, 0x38, 0x6f, 0x1e, 0x87, 0x33, 0x6f, 0x9e, 0x04, 0xa0,
	0xf9, 0xad, 0x59, 0x28, 0x2d, 0x8d, 0x64, 0xa1, 0x56, 0xdb, 0x7c, 0x0d, 0xf1, 0x27, 0x29, 0xb0,
	0x65, 0x53, 0x48, 0x5a, 0x6c, 0x04, 0xea, 0xcd, 0x3a, 0x0b, 0x66, 0xc1, 0x3c, 0x2c, 0x86, 0x98,
	0xbd, 0x86, 0xb8, 0xb1, 0xa4, 0x6c, 0x34, 0x0b, 0xe7, 0xe9, 0xf2, 0x74, 0xa1, 0xd5, 0x76, 0x61,
	0xaf, 0x15, 0x0e, 0xcf, 0x7f, 0x8d, 0xe0, 0x6c, 0xa5, 0x14, 0x36, 0xa2, 0xc0, 0xef, 0xf7, 0xd8,
	0x9a, 0xa3, 0xe5, 0x18, 0x44, 0x06, 0x75, 0x9d, 0x8d, 0x08, 0xa7, 0xb3, 0xe5, 0xef, 0x90, 0x3b,
	0x7e, 0xe8, 0xf8, 0x7d, 0xcc, 0x5e, 0x01, 0x28, 0x8d, 0x0f, 0x37, 0xf2, 0xdb, 0x46, 0x3c, 0x66,
	0x11, 0x65, 0x3d, 0x84, 0xcd, 0x20, 0xed, 0xa2, 0x2f, 0xb6, 0x6c, 0x4c, 0x04, 0x1f, 0x62, 0x73,
	0x78, 0xe6, 0xaa, 0x5d, 0xc9, 0xba, 0xae, 0x8c, 0x2d, 0x73, 0x42, 0xac, 0x43, 0x98, 0xe5, 0x30,
	0xd9, 0xf1, 0xd6, 0x5c, 0x37, 0x46, 0xff, 0xb0, 0xb4, 0x31, 0xd1, 0xf6, 0x30, 0xf6, 0x16, 0xc6,
	0xd8, 0x18, 0x5d, 0x61, 0x9b, 0x25, 0x24, 0x08, 0x90, 0x20, 0x94, 0x2f, 0xfa, 0x54, 0xfe, 0x33,
	0x80, 0xf4, 0xab, 0x34, 0xd8, 0x2b, 0xd2, 0x4f, 0x1d, 0x78, 0x53, 0xcf, 0x20, 0x2d, 0x79, 0x23,
	0x2a, 0xc1, 0x0d, 0x6e, 0xd6, 0x9d, 0x20, 0x3e, 0x64, 0x67, 0xb7, 0x6f, 0x77, 0xb3, 0x3b, 0x65,
	0x3c, 0xc4, 0x56, 0xe8, 0x22, 0x9a, 0xdd, 0x89, 0xe3, 0x43, 0xf9, 0x07, 0x88, 0xa9, 0xb3, 0x7f,
	0x36, 0x70, 0x01, 0x61, 0x25, 0x1e, 0xbb, 0x87, 0xed, 0x91, 0x65, 0x30, 0x2e, 0x65, 0x5d, 0xf3,
	0x46, 0xd0, 0x6b, 0x93, 0xa2, 0x0f, 0xf3, 0x77, 0x10, 0xd9, 0x9d, 0xb3, 0x73, 0x18, 0x0d, 0x4b,
	0x1d, 0xb9, 0x75, 0x72, 0x21, 0x34, 0x15, 0x39, 0x2d, 0xe8, 0x9c, 0xff, 0x0e, 0x20, 0x29, 0xb0,
	0x55, 0xb2, 0x69, 0xf1, 0xa8, 0x17, 0x9e, 0x43, 0x8c, 0x5a, 0xcb, 0xfe, 0xb6, 0x0b, 0xec, 0x0d,
	0x5e, 0x96, 0xa8, 0x0c, 0xba, 0x2e, 0x92, 0x62, 0x88, 0x87, 0x31, 0xa2, 0xff, 0xb8, 0x27, 0x3e,
	0x70, 0xcf, 0x25, 0x9c, 0x5b, 0x39, 0x56, 0x4a, 0xed, 0x2a, 0x14, 0x4f, 0xab, 0x3f, 0x40, 0xf3,
	0x37, 0x10, 0xae, 0xca, 0xbb, 0x63, 0xcd, 0x2e, 0xff, 0x04, 0x10, 0x15, 0xfc, 0xd6, 0xb0, 0x65,
	0x6f, 0xf7, 0x6b, 0xb7, 0x6c, 0xc6, 0xc8, 0x01, 0x7b, 0x9f, 0xc0, 0xf4, 0x8c, 0xb0, 0x41, 0x85,
	0x05, 0xa4, 0x5d, 0xc6, 0xba, 0x82, 0x5d, 0x50, 0xd6, 0x33, 0xc8, 0x21, 0xff, 0x25, 0x44, 0x1f,
	0x65, 0xd5, 0xb0, 0xa7, 0xaf, 0x6d, 0x0a, 0xc3, 0xb1, 0x65, 0x97, 0x30, 0xb9, 0xda, 0xdd, 0xb7,
	0x06, 0xf5, 0x67, 0xc3, 0x0d, 0x32, 0x2f, 0xb7, 0xc7, 0x7b, 0x01, 0xf1, 0x0d, 0xf2, 0x07, 0xf4,
	0xeb, 0x24, 0xae, 0xdb, 0xf2, 0x6e, 0x7b, 0x42, 0xbf, 0x82, 0xf7, 0x7f, 0x03, 0x00, 0x00, 0xff,
	0xff, 0x9f, 0x89, 0x93, 0x8a, 0x18, 0x04, 0x00, 0x00,
}