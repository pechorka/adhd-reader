// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: api/api.proto

package api

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AdhdReaderServiceClient is the client API for AdhdReaderService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AdhdReaderServiceClient interface {
	SetChunkSize(ctx context.Context, in *SetChunkSizeRequest, opts ...grpc.CallOption) (*SetChunkSizeResponse, error)
	AddText(ctx context.Context, in *AddTextRequest, opts ...grpc.CallOption) (*AddTextResponse, error)
	ListTexts(ctx context.Context, in *ListTextsRequest, opts ...grpc.CallOption) (*ListTextsResponse, error)
	SelectText(ctx context.Context, in *SelectTextRequest, opts ...grpc.CallOption) (*SelectTextResponse, error)
	SetPage(ctx context.Context, in *SetPageRequest, opts ...grpc.CallOption) (*SetPageResponse, error)
	NextChunk(ctx context.Context, in *NextChunkRequest, opts ...grpc.CallOption) (*NextChunkResponse, error)
	PrevChunk(ctx context.Context, in *PrevChunkRequest, opts ...grpc.CallOption) (*PrevChunkResponse, error)
}

type adhdReaderServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAdhdReaderServiceClient(cc grpc.ClientConnInterface) AdhdReaderServiceClient {
	return &adhdReaderServiceClient{cc}
}

func (c *adhdReaderServiceClient) SetChunkSize(ctx context.Context, in *SetChunkSizeRequest, opts ...grpc.CallOption) (*SetChunkSizeResponse, error) {
	out := new(SetChunkSizeResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/SetChunkSize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adhdReaderServiceClient) AddText(ctx context.Context, in *AddTextRequest, opts ...grpc.CallOption) (*AddTextResponse, error) {
	out := new(AddTextResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/AddText", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adhdReaderServiceClient) ListTexts(ctx context.Context, in *ListTextsRequest, opts ...grpc.CallOption) (*ListTextsResponse, error) {
	out := new(ListTextsResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/ListTexts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adhdReaderServiceClient) SelectText(ctx context.Context, in *SelectTextRequest, opts ...grpc.CallOption) (*SelectTextResponse, error) {
	out := new(SelectTextResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/SelectText", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adhdReaderServiceClient) SetPage(ctx context.Context, in *SetPageRequest, opts ...grpc.CallOption) (*SetPageResponse, error) {
	out := new(SetPageResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/SetPage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adhdReaderServiceClient) NextChunk(ctx context.Context, in *NextChunkRequest, opts ...grpc.CallOption) (*NextChunkResponse, error) {
	out := new(NextChunkResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/NextChunk", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adhdReaderServiceClient) PrevChunk(ctx context.Context, in *PrevChunkRequest, opts ...grpc.CallOption) (*PrevChunkResponse, error) {
	out := new(PrevChunkResponse)
	err := c.cc.Invoke(ctx, "/api.AdhdReaderService/PrevChunk", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AdhdReaderServiceServer is the server API for AdhdReaderService service.
// All implementations must embed UnimplementedAdhdReaderServiceServer
// for forward compatibility
type AdhdReaderServiceServer interface {
	SetChunkSize(context.Context, *SetChunkSizeRequest) (*SetChunkSizeResponse, error)
	AddText(context.Context, *AddTextRequest) (*AddTextResponse, error)
	ListTexts(context.Context, *ListTextsRequest) (*ListTextsResponse, error)
	SelectText(context.Context, *SelectTextRequest) (*SelectTextResponse, error)
	SetPage(context.Context, *SetPageRequest) (*SetPageResponse, error)
	NextChunk(context.Context, *NextChunkRequest) (*NextChunkResponse, error)
	PrevChunk(context.Context, *PrevChunkRequest) (*PrevChunkResponse, error)
	mustEmbedUnimplementedAdhdReaderServiceServer()
}

// UnimplementedAdhdReaderServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAdhdReaderServiceServer struct {
}

func (UnimplementedAdhdReaderServiceServer) SetChunkSize(context.Context, *SetChunkSizeRequest) (*SetChunkSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetChunkSize not implemented")
}
func (UnimplementedAdhdReaderServiceServer) AddText(context.Context, *AddTextRequest) (*AddTextResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddText not implemented")
}
func (UnimplementedAdhdReaderServiceServer) ListTexts(context.Context, *ListTextsRequest) (*ListTextsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTexts not implemented")
}
func (UnimplementedAdhdReaderServiceServer) SelectText(context.Context, *SelectTextRequest) (*SelectTextResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SelectText not implemented")
}
func (UnimplementedAdhdReaderServiceServer) SetPage(context.Context, *SetPageRequest) (*SetPageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetPage not implemented")
}
func (UnimplementedAdhdReaderServiceServer) NextChunk(context.Context, *NextChunkRequest) (*NextChunkResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NextChunk not implemented")
}
func (UnimplementedAdhdReaderServiceServer) PrevChunk(context.Context, *PrevChunkRequest) (*PrevChunkResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PrevChunk not implemented")
}
func (UnimplementedAdhdReaderServiceServer) mustEmbedUnimplementedAdhdReaderServiceServer() {}

// UnsafeAdhdReaderServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AdhdReaderServiceServer will
// result in compilation errors.
type UnsafeAdhdReaderServiceServer interface {
	mustEmbedUnimplementedAdhdReaderServiceServer()
}

func RegisterAdhdReaderServiceServer(s grpc.ServiceRegistrar, srv AdhdReaderServiceServer) {
	s.RegisterService(&AdhdReaderService_ServiceDesc, srv)
}

func _AdhdReaderService_SetChunkSize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetChunkSizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).SetChunkSize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/SetChunkSize",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).SetChunkSize(ctx, req.(*SetChunkSizeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdhdReaderService_AddText_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddTextRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).AddText(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/AddText",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).AddText(ctx, req.(*AddTextRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdhdReaderService_ListTexts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTextsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).ListTexts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/ListTexts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).ListTexts(ctx, req.(*ListTextsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdhdReaderService_SelectText_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SelectTextRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).SelectText(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/SelectText",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).SelectText(ctx, req.(*SelectTextRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdhdReaderService_SetPage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetPageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).SetPage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/SetPage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).SetPage(ctx, req.(*SetPageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdhdReaderService_NextChunk_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NextChunkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).NextChunk(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/NextChunk",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).NextChunk(ctx, req.(*NextChunkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdhdReaderService_PrevChunk_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PrevChunkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdhdReaderServiceServer).PrevChunk(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.AdhdReaderService/PrevChunk",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdhdReaderServiceServer).PrevChunk(ctx, req.(*PrevChunkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// AdhdReaderService_ServiceDesc is the grpc.ServiceDesc for AdhdReaderService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AdhdReaderService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.AdhdReaderService",
	HandlerType: (*AdhdReaderServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetChunkSize",
			Handler:    _AdhdReaderService_SetChunkSize_Handler,
		},
		{
			MethodName: "AddText",
			Handler:    _AdhdReaderService_AddText_Handler,
		},
		{
			MethodName: "ListTexts",
			Handler:    _AdhdReaderService_ListTexts_Handler,
		},
		{
			MethodName: "SelectText",
			Handler:    _AdhdReaderService_SelectText_Handler,
		},
		{
			MethodName: "SetPage",
			Handler:    _AdhdReaderService_SetPage_Handler,
		},
		{
			MethodName: "NextChunk",
			Handler:    _AdhdReaderService_NextChunk_Handler,
		},
		{
			MethodName: "PrevChunk",
			Handler:    _AdhdReaderService_PrevChunk_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/api.proto",
}