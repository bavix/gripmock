package app

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

type StreamType int

const (
	StreamTypeUnary StreamType = iota
	StreamTypeServer
	StreamTypeClient
	StreamTypeBidi
)

type FallbackError struct {
	err        error
	request    *dynamicpb.Message
	requests   []*dynamicpb.Message
	streamType StreamType
}

func (e *FallbackError) Error() string {
	return e.err.Error()
}

func (e *FallbackError) Unwrap() error {
	return e.err
}

func (e *FallbackError) GRPCStatus() *status.Status {
	if e.streamType == StreamTypeBidi {
		return status.New(codes.NotFound, e.err.Error())
	}
	return status.Convert(e.err)
}

func newUnaryFallbackError(err error) *FallbackError {
	return &FallbackError{err: err, streamType: StreamTypeUnary}
}

func newServerStreamFallbackError(err error, req *dynamicpb.Message) *FallbackError {
	return &FallbackError{err: err, request: req, streamType: StreamTypeServer}
}

func newClientStreamFallbackError(err error, reqs []*dynamicpb.Message) *FallbackError {
	return &FallbackError{err: err, requests: reqs, streamType: StreamTypeClient}
}

func newBidiStreamFallbackError(err error, reqs []*dynamicpb.Message) *FallbackError {
	return &FallbackError{err: err, requests: reqs, streamType: StreamTypeBidi}
}
