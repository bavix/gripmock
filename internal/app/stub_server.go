package app

import (
	"context"
	"github.com/bavix/gripmock/pkg/api/stubs"
	"github.com/bavix/gripmock/pkg/storage"
	"github.com/bavix/gripmock/pkg/yaml2json"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type StubsServer struct {
	stubs     *storage.StubStorage
	convertor *yaml2json.Convertor
	titler    cases.Caser
}

func NewStubsServer() *StubsServer {
	return &StubsServer{
		stubs:     storage.New(),
		convertor: yaml2json.New(),
		titler:    cases.Title(language.English, cases.NoLower),
	}
}

// deprecated code
type findStubPayload struct {
	Service string      `json:"service"`
	Method  string      `json:"method"`
	Data    interface{} `json:"data"`
}

func (s *StubsServer) AddStub(ctx context.Context, req api.AddStubReq) (api.AddStubOK, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StubsServer) DeleteStubByID(ctx context.Context, params api.DeleteStubByIDParams) (api.DeleteStubByIDRes, error) {
	panic("implement me")
}

func (s *StubsServer) ListStubs(ctx context.Context) (api.StubList, error) {
	panic("implement me")
}

func (s *StubsServer) PurgeStubs(ctx context.Context) error {
	s.stubs.Purge()

	return nil
}

func (s *StubsServer) SearchStubs(ctx context.Context, req *api.SearchRequest) (*api.SearchResponse, error) {
	output, err := findStub(s.stubs, &findStubPayload{
		Service: req.Service,
		Method:  s.titler.String(req.Method),
		Data:    req.Data,
	})

	if err != nil {
		return nil, err
	}

	return &api.SearchResponse{
		Data:  output.Data,
		Error: api.NewOptString(output.Error),
		Code:  api.NewOptUint32(output.Code),
	}, nil
}
