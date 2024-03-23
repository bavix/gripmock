package storage

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"google.golang.org/grpc/codes"
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrMethodNotFound  = errors.New("method not found")
)

type Stub struct {
	ID      *uuid.UUID  `json:"id,omitempty"`
	Service string      `json:"service"`
	Method  string      `json:"method"`
	Headers InputHeader `json:"headers"`
	Input   InputData   `json:"input"`
	Output  Output      `json:"output"`
}

func (s *Stub) GetID() uuid.UUID {
	if s.ID == nil {
		id := uuid.New()
		s.ID = &id
	}

	return *s.ID
}

type InputInterface interface {
	GetEquals() map[string]interface{}
	GetContains() map[string]interface{}
	GetMatches() map[string]interface{}
}

type InputData struct {
	IgnoreArrayOrder bool                   `json:"ignoreArrayOrder,omitempty"`
	Equals           map[string]interface{} `json:"equals"`
	Contains         map[string]interface{} `json:"contains"`
	Matches          map[string]interface{} `json:"matches"`
}

func (i InputData) GetEquals() map[string]interface{} {
	return i.Equals
}

func (i InputData) GetContains() map[string]interface{} {
	return i.Contains
}

func (i InputData) GetMatches() map[string]interface{} {
	return i.Matches
}

type InputHeader struct {
	Equals   map[string]interface{} `json:"equals"`
	Contains map[string]interface{} `json:"contains"`
	Matches  map[string]interface{} `json:"matches"`
}

func (i InputHeader) GetEquals() map[string]interface{} {
	return i.Equals
}

func (i InputHeader) GetContains() map[string]interface{} {
	return i.Contains
}

func (i InputHeader) GetMatches() map[string]interface{} {
	return i.Matches
}

type Output struct {
	Headers map[string]string      `json:"headers"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error"`
	Code    *codes.Code            `json:"code,omitempty"`
}

type storage struct {
	ID      uuid.UUID
	Headers InputHeader
	Input   InputData
	Output  Output
}

func (s *storage) CountHeaders() int {
	return len(s.Headers.Equals) + len(s.Headers.Matches) + len(s.Headers.Contains)
}

func (s *storage) CheckHeaders() bool {
	return s.CountHeaders() > 0
}

type StubStorage struct {
	mu    sync.Mutex
	used  map[uuid.UUID]struct{}
	db    *memdb.MemDB
	total int64
}

func New() (*StubStorage, error) {
	db, err := memdb.NewMemDB(&memdb.DBSchema{Tables: schema()})
	if err != nil {
		return nil, err
	}

	return &StubStorage{db: db, used: map[uuid.UUID]struct{}{}}, nil
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

		delete(r.used, arg)
	}

	atomic.AddInt64(&r.total, -total)
}

func (r *StubStorage) Purge() {
	txn := r.db.Txn(true)
	defer txn.Commit()

	n, _ := txn.DeleteAll(TableName, IDField)
	r.used = map[uuid.UUID]struct{}{}

	atomic.AddInt64(&r.total, -int64(n))
}

func (r *StubStorage) ItemsBy(service, method string, ID *uuid.UUID) ([]storage, error) {
	txn := r.db.Txn(true)
	defer txn.Commit()

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

	var it memdb.ResultIterator

	if ID == nil {
		it, err = txn.Get(TableName, ServiceMethodField, service, method)
	} else {
		it, err = txn.Get(TableName, IDField, ID)
	}

	if err != nil {
		return nil, err
	}

	var result []storage

	for obj := it.Next(); obj != nil; obj = it.Next() {
		stub, _ := obj.(*Stub)

		s := storage{
			ID:      stub.GetID(),
			Headers: stub.Headers,
			Input:   stub.Input,
			Output:  stub.Output,
		}

		result = append(result, s)
	}

	slices.SortFunc(result, func(a, b storage) int {
		return b.CountHeaders() - a.CountHeaders()
	})

	return result, nil
}

func (r *StubStorage) Used() []Stub {
	txn := r.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(TableName, IDField)
	if err != nil {
		return nil
	}

	it := memdb.NewFilterIterator(iter, func(raw interface{}) bool {
		obj, ok := raw.(*Stub)
		if !ok {
			return true
		}

		_, ok = r.used[obj.GetID()]

		return !ok
	})

	result := make([]Stub, 0, atomic.LoadInt64(&r.total))

	for obj := it.Next(); obj != nil; obj = it.Next() {
		stub, _ := obj.(*Stub)

		result = append(result, *stub)
	}

	return result
}

func (r *StubStorage) Unused() []Stub {
	txn := r.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(TableName, IDField)
	if err != nil {
		return nil
	}

	it := memdb.NewFilterIterator(iter, func(raw interface{}) bool {
		obj, ok := raw.(*Stub)
		if !ok {
			return true
		}

		_, ok = r.used[obj.GetID()]

		return ok
	})

	result := make([]Stub, 0, atomic.LoadInt64(&r.total))

	for obj := it.Next(); obj != nil; obj = it.Next() {
		stub, _ := obj.(*Stub)

		result = append(result, *stub)
	}

	return result
}

func (r *StubStorage) MarkUsed(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.used[id] = struct{}{}
}

func (r *StubStorage) FindByID(id uuid.UUID) *Stub {
	txn := r.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(TableName, IDField, id)
	if err != nil {
		return nil
	}

	for obj := it.Next(); obj != nil; {
		stub, _ := obj.(*Stub)

		return stub
	}

	return nil
}

func (r *StubStorage) Stubs() []Stub {
	txn := r.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(TableName, IDField)
	if err != nil {
		return nil
	}

	result := make([]Stub, 0, atomic.LoadInt64(&r.total))

	for obj := it.Next(); obj != nil; obj = it.Next() {
		stub, _ := obj.(*Stub)

		result = append(result, *stub)
	}

	return result
}
