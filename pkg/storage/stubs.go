package storage

import (
	"errors"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"google.golang.org/grpc/codes"
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

func (s *Stub) GetID() uuid.UUID {
	if s.ID == nil {
		id := uuid.New()
		s.ID = &id
	}

	return *s.ID
}

type Input struct {
	Equals   map[string]interface{} `json:"equals"`
	Contains map[string]interface{} `json:"contains"`
	Matches  map[string]interface{} `json:"matches"`
}

type Output struct {
	Data  map[string]interface{} `json:"data"`
	Error string                 `json:"error"`
	Code  *codes.Code            `json:"code,omitempty"`
}

type storage struct {
	ID     uuid.UUID
	Input  Input
	Output Output
}

type StubStorage struct {
	db    *memdb.MemDB
	total int64
}

func New() (*StubStorage, error) {
	db, err := memdb.NewMemDB(&memdb.DBSchema{Tables: schema()})
	if err != nil {
		return nil, err
	}

	return &StubStorage{db: db}, nil
}

func (r *StubStorage) Add(stubs ...*Stub) []uuid.UUID {
	txn := r.db.Txn(true)

	result := make([]uuid.UUID, 0, len(stubs))

	for _, stub := range stubs {
		stub.GetID() // init id if not exists

		err := txn.Insert(TableName, stub)
		if err != nil {
			txn.Abort()

			return nil
		}

		result = append(result, stub.GetID())
	}

	atomic.AddInt64(&r.total, int64(len(result)))

	txn.Commit()

	return result
}

func (r *StubStorage) Delete(args ...uuid.UUID) {
	txn := r.db.Txn(true)
	defer txn.Commit()

	var total int64
	for _, arg := range args {
		n, _ := txn.DeleteAll(TableName, IDField, arg)

		total += int64(n)
	}

	atomic.AddInt64(&r.total, -total)
}

func (r *StubStorage) Purge() {
	txn := r.db.Txn(true)
	defer txn.Commit()

	n, _ := txn.DeleteAll(TableName, IDField)

	atomic.AddInt64(&r.total, -int64(n))
}

func (r *StubStorage) ItemsBy(service, method string) ([]storage, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	// Support for backward compatibility. Someday it will be redone...
	first, err := txn.First(TableName, ServiceField, service)
	if err != nil || first == nil {
		return nil, ErrServiceNotFound
	}

	// Support for backward compatibility. Someday it will be redone...
	first, err = txn.First(TableName, ServiceMethodField)
	if err != nil || first == nil {
		return nil, ErrMethodNotFound
	}

	it, err := txn.Get(TableName, ServiceMethodField, service, method)
	if err != nil {
		return nil, err
	}

	var result []storage

	for obj := it.Next(); obj != nil; obj = it.Next() {
		stub := obj.(*Stub)

		result = append(result, storage{
			ID:     stub.GetID(),
			Input:  stub.Input,
			Output: stub.Output,
		})
	}

	return result, nil
}

func (r *StubStorage) Stubs() []Stub {
	txn := r.db.Txn(false)
	defer txn.Abort()

	result := make([]Stub, 0, atomic.LoadInt64(&r.total))

	it, err := txn.Get(TableName, IDField)
	if err != nil {
		return nil
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		stub := obj.(*Stub)

		result = append(result, *stub)
	}

	return result
}
