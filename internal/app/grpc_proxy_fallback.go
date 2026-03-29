package app

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

type unaryStubMissError struct {
	err error
}

func (e *unaryStubMissError) Error() string {
	return e.err.Error()
}

func (e *unaryStubMissError) Unwrap() error {
	return e.err
}

func (e *unaryStubMissError) GRPCStatus() *status.Status {
	return status.Convert(e.err)
}

type serverStreamFallbackError struct {
	err     error
	request *dynamicpb.Message
}

func (e *serverStreamFallbackError) Error() string {
	return e.err.Error()
}

func (e *serverStreamFallbackError) Unwrap() error {
	return e.err
}

func (e *serverStreamFallbackError) GRPCStatus() *status.Status {
	return status.Convert(e.err)
}

type clientStreamFallbackError struct {
	err      error
	requests []*dynamicpb.Message
}

func (e *clientStreamFallbackError) Error() string {
	return e.err.Error()
}

func (e *clientStreamFallbackError) Unwrap() error {
	return e.err
}

func (e *clientStreamFallbackError) GRPCStatus() *status.Status {
	return status.Convert(e.err)
}

type bidiStreamFallbackError struct {
	err      error
	requests []*dynamicpb.Message
}

func (e *bidiStreamFallbackError) Error() string {
	return e.err.Error()
}

func (e *bidiStreamFallbackError) Unwrap() error {
	return e.err
}

func (e *bidiStreamFallbackError) GRPCStatus() *status.Status {
	return status.New(codes.NotFound, e.err.Error())
}
