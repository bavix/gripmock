package stuber

type searcherOptions struct {
	lookupProvider searcherLookupProvider
	lookupFactory  searcherLookupFactory
}

// newSearcher creates a new searcher instance.
func newSearcher() *searcher {
	return newSearcherWithOptions(searcherOptions{})
}

func newSearcherWithOptions(options searcherOptions) *searcher {
	lookupProvider := options.lookupProvider
	lookupFactory := options.lookupFactory

	if lookupProvider == nil && (lookupFactory.newID != nil || lookupFactory.newService != nil || lookupFactory.newMethod != nil) {
		lookupProvider = lookupFactory
	}

	if lookupProvider == nil {
		lookupProvider = defaultSearcherLookupFactory()
	}

	storage := newStorageWithInternal()

	return &searcher{
		storage:         storage,
		internalStorage: storage.Internal(),
		stubCallCount:   make(map[callCountKey]int),
		lookupProvider:  lookupProvider,
		lookupCache:     make(map[string]*searcherLookup),
	}
}
