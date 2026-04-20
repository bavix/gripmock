package stuber

import (
	"iter"
	"slices"
	"sort"
)

const (
	ListSortPriorityDesc = "priority_desc"
	ListSortPriorityAsc  = "priority_asc"
	ListSortServiceAsc   = "service_asc"
	ListSortMethodAsc    = "method_asc"
)

// ListOptions controls filtering, sorting and pagination for stubs listing.
type ListOptions struct {
	Source  string
	Service string
	Method  string

	Session    string
	SessionSet bool

	Limit  int
	Offset int
	Sort   string
}

// List returns filtered stubs and total before pagination.
func (b *Budgerigar) List(options ListOptions) ([]*Stub, int) {
	filtered := filterStubs(b.searcher.storage.values(), options)

	sortStubs(filtered, options.Sort)

	total := len(filtered)
	filtered = paginateStubs(filtered, options)

	return filtered, total
}

func filterStubs(stubs iter.Seq[*Stub], options ListOptions) []*Stub {
	seq := stubs

	if options.Source != "" {
		source := options.Source
		seq = whereStubs(seq, func(stub *Stub) bool {
			return stub.Source == source
		})
	}

	if options.Service != "" {
		service := options.Service
		seq = whereStubs(seq, func(stub *Stub) bool {
			return stub.Service == service
		})
	}

	if options.Method != "" {
		method := options.Method
		seq = whereStubs(seq, func(stub *Stub) bool {
			return stub.Method == method
		})
	}

	if options.SessionSet {
		session := options.Session
		seq = whereStubs(seq, func(stub *Stub) bool {
			return stub.Session == session
		})
	}

	filtered := slices.Collect(seq)
	if filtered == nil {
		return []*Stub{}
	}

	return filtered
}

func whereStubs(seq iter.Seq[*Stub], keep func(*Stub) bool) iter.Seq[*Stub] {
	return func(yield func(*Stub) bool) {
		for stub := range seq {
			if !keep(stub) {
				continue
			}

			if !yield(stub) {
				return
			}
		}
	}
}

func paginateStubs(stubs []*Stub, options ListOptions) []*Stub {
	offset := min(max(options.Offset, 0), len(stubs))
	stubs = stubs[offset:]

	if options.Limit > 0 {
		stubs = stubs[:min(options.Limit, len(stubs))]
	}

	return stubs
}

func sortStubs(stubs []*Stub, mode string) {
	less := func(i, j int) bool {
		return stubs[i].Priority > stubs[j].Priority
	}

	switch mode {
	case ListSortPriorityAsc:
		less = func(i, j int) bool {
			return stubs[i].Priority < stubs[j].Priority
		}
	case ListSortServiceAsc:
		less = func(i, j int) bool {
			if stubs[i].Service == stubs[j].Service {
				return stubs[i].Method < stubs[j].Method
			}

			return stubs[i].Service < stubs[j].Service
		}
	case ListSortMethodAsc:
		less = func(i, j int) bool {
			if stubs[i].Method == stubs[j].Method {
				return stubs[i].Service < stubs[j].Service
			}

			return stubs[i].Method < stubs[j].Method
		}
	}

	sort.SliceStable(stubs, less)
}
