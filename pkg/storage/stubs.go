package storage

import (
	"errors"
	"github.com/google/uuid"
	"sync"
)

var ErrServiceNotFound = errors.New("service not found")
var ErrMethodNotFound = errors.New("method not found")

type Stub struct {
	ID      *uuid.UUID `json:"id,omitempty"`
	Service string     `json:"service"`
	Method  string     `json:"method"`
	Input   Input      `json:"input"`
	Output  Output     `json:"output"`
}

type Input struct {
	Equals   map[string]interface{} `json:"equals"`
	Contains map[string]interface{} `json:"contains"`
	Matches  map[string]interface{} `json:"matches"`
}

type Output struct {
	Data  map[string]interface{} `json:"data"`
	Error string                 `json:"error"`
}

type storage struct {
	ID     uuid.UUID
	Input  Input
	Output Output
}

type StubStorage struct {
	mu    sync.RWMutex
	items map[string]map[string][]storage
	total uint64
}

func New() *StubStorage {
	return &StubStorage{
		items: make(map[string]map[string][]storage),
	}
}

func (r *StubStorage) Add(stubs ...*Stub) []uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]uuid.UUID, 0, len(stubs))

	for _, stub := range stubs {
		if _, ok := r.items[stub.Service]; !ok {
			r.items[stub.Service] = make(map[string][]storage, 1)
		}

		r.items[stub.Service][stub.Method] = append(r.items[stub.Service][stub.Method], storage{
			ID:     stub.GetID(),
			Input:  stub.Input,
			Output: stub.Output,
		})

		result = append(result, stub.GetID())

		r.total++
	}

	return result
}

func (r *StubStorage) Delete(_ ...uuid.UUID) {
	r.total-- // fixme
}

func (r *StubStorage) Purge() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = map[string]map[string][]storage{}
	r.total = 0
}

func (r *StubStorage) ItemsBy(service, method string) ([]storage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.items[service]; !ok {
		return nil, ErrServiceNotFound
	}

	if _, ok := r.items[service][method]; !ok {
		return nil, ErrMethodNotFound
	}

	return r.items[service][method], nil
}

func (r *StubStorage) Stubs() []Stub {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]Stub, 0, r.total)

	for service, methods := range r.items {
		for method, storages := range methods {
			for _, datum := range storages {
				results = append(results, Stub{
					ID:      &datum.ID,
					Service: service,
					Method:  method,
					Input:   datum.Input,
					Output:  datum.Output,
				})
			}
		}
	}

	return results
}

func (s *Stub) GetID() uuid.UUID {
	if s.ID == nil {
		id := uuid.New()
		s.ID = &id
	}

	return *s.ID
}
