// Code generated by ogen, DO NOT EDIT.

package api

import (
	"context"
)

// Handler handles operations described by OpenAPI v3 specification.
type Handler interface {
	// AddStub implements addStub operation.
	//
	// Add a new stub to the store.
	//
	// POST /stubs
	AddStub(ctx context.Context, req AddStubReq) (AddStubOK, error)
	// DeleteStubByID implements deleteStubByID operation.
	//
	// The method removes the stub by ID.
	//
	// DELETE /stubs/{uuid}
	DeleteStubByID(ctx context.Context, params DeleteStubByIDParams) (DeleteStubByIDRes, error)
	// ListStubs implements listStubs operation.
	//
	// The list of stubs is required to view all added stubs.
	//
	// GET /stubs
	ListStubs(ctx context.Context) (StubList, error)
	// PurgeStubs implements purgeStubs operation.
	//
	// Completely clears the stub storage.
	//
	// DELETE /stubs
	PurgeStubs(ctx context.Context) error
	// SearchStubs implements searchStubs operation.
	//
	// Performs a search for a stub by the given conditions.
	//
	// POST /stubs/search
	SearchStubs(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
}

// Server implements http server based on OpenAPI v3 specification and
// calls Handler to handle requests.
type Server struct {
	h Handler
	baseServer
}

// NewServer creates new Server.
func NewServer(h Handler, opts ...ServerOption) (*Server, error) {
	s, err := newServerConfig(opts...).baseServer()
	if err != nil {
		return nil, err
	}
	return &Server{
		h:          h,
		baseServer: s,
	}, nil
}
