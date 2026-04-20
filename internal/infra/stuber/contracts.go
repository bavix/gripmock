package stuber

// ListContract defines list/filter/sort contract for stubs.
type ListContract interface {
	List(options ListOptions) ([]*Stub, int)
}

// DumpContract defines dump/export contract for stubs.
type DumpContract interface {
	DumpToDir(outDir string, stubs []*Stub, format string) (int, error)
}
