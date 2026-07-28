package stuber

import (
	"bytes"
	"iter"
	"slices"
	"sort"
	"strings"
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

	// Query is a case-insensitive substring matched against the stub's
	// service, method and ID. Empty means no text filter.
	Query string

	// Matchers keeps only stubs whose input declares at least one of the given
	// matcher kinds (equals/contains/matches/glob/anyOf) — OR semantics. Empty
	// means no matcher-kind filter.
	Matchers []string

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

//nolint:cyclop
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

	if options.Query != "" {
		q := strings.ToLower(options.Query)
		seq = whereStubs(seq, func(stub *Stub) bool {
			return strings.Contains(strings.ToLower(stub.Service), q) ||
				strings.Contains(strings.ToLower(stub.Method), q) ||
				strings.Contains(strings.ToLower(stub.ID.String()), q)
		})
	}

	if len(options.Matchers) > 0 {
		kinds := options.Matchers
		seq = whereStubs(seq, func(stub *Stub) bool {
			for _, kind := range kinds {
				if stubHasMatcherKind(stub, kind) {
					return true
				}
			}

			return false
		})
	}

	filtered := slices.Collect(seq)
	if filtered == nil {
		return []*Stub{}
	}

	return filtered
}

// stubHasMatcherKind reports whether the stub's input (unary Input, stream
// Inputs, or their AnyOf alternatives) declares the given matcher kind.
func stubHasMatcherKind(stub *Stub, kind string) bool {
	if inputHasKind(stub.Input, kind) {
		return true
	}

	for _, in := range stub.Inputs {
		if inputHasKind(in, kind) {
			return true
		}
	}

	return false
}

func inputHasKind(in InputData, kind string) bool {
	switch kind {
	case "equals":
		return len(in.Equals) > 0
	case "contains":
		return len(in.Contains) > 0
	case "matches":
		return len(in.Matches) > 0
	case "glob":
		return len(in.Glob) > 0
	case "anyOf":
		return len(in.AnyOf) > 0
	default:
		return false
	}
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

// SortStubs sorts an already-materialized stub slice in place using the same
// modes as List (ListSort* constants). Exposed for callers that filter stubs
// outside the storage iterator (e.g. the MCP used/unused sets) yet want ordering
// consistent with the REST /stubs listing.
func SortStubs(stubs []*Stub, mode string) {
	sortStubs(stubs, mode)
}

func sortStubs(stubs []*Stub, mode string) {
	// Every mode ends in an ID tiebreak so the order is a TOTAL order. Input comes
	// from values(), which ranges a Go map (random order each call); without the
	// tiebreak, stubs tying on the sort key keep that random order and pagination
	// duplicates/drops rows across independently-sorted pages.
	byID := func(i, j int) bool {
		return bytes.Compare(stubs[i].ID[:], stubs[j].ID[:]) < 0
	}

	less := func(i, j int) bool {
		if stubs[i].Priority != stubs[j].Priority {
			return stubs[i].Priority > stubs[j].Priority
		}

		return byID(i, j)
	}

	switch mode {
	case ListSortPriorityAsc:
		less = func(i, j int) bool {
			if stubs[i].Priority != stubs[j].Priority {
				return stubs[i].Priority < stubs[j].Priority
			}

			return byID(i, j)
		}
	case ListSortServiceAsc:
		less = func(i, j int) bool {
			if stubs[i].Service != stubs[j].Service {
				return stubs[i].Service < stubs[j].Service
			}

			if stubs[i].Method != stubs[j].Method {
				return stubs[i].Method < stubs[j].Method
			}

			return byID(i, j)
		}
	case ListSortMethodAsc:
		less = func(i, j int) bool {
			if stubs[i].Method != stubs[j].Method {
				return stubs[i].Method < stubs[j].Method
			}

			if stubs[i].Service != stubs[j].Service {
				return stubs[i].Service < stubs[j].Service
			}

			return byID(i, j)
		}
	}

	sort.SliceStable(stubs, less)
}
