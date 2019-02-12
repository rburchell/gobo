package search

// ResultIdentifier is a way of uniquely identifying a result in the system.
// It can be thought of as being somewhat similar to a primary key in a database
// table.
type ResultIdentifier int64

// A TypedResult is a response to a typed query.
// For instance, if you make a type query of 'year', then you will get back a
// result of '2007' if a document has been tagged with a value of 2007 for year.
type TypedResult struct {
	TypeValue string
	ID        ResultIdentifier
}

// An Index is a thing that can be queried by this package.
//
// Queries are first costed (using the Cost* functions), then executed in the
// cheapest manner possible using that cost evaluation, by calls to the Query*
// functions.
//
// The index can also be filtered, which can help make the Query* functions
// cheaper to implement (by eliminating results which are not in the filtered
// map). Ergo, though it is not mentioned explicitly, the Query* functions are
// _expected_ to filter their results in order to provide decent performance.
//
// Query results are expected to return a channel of results which can be
// streamed through other queries.
//
// All functions in an Index must be concurrent-safe; they may be called from
// any goroutine.
//
// Note that while this interface does not force it, it is wise to canonicalize
// tags, such that querying for "Cat" and "cat" find the same results (and
// similar).
type Index interface {
	// Return all results.
	QueryAll() []ResultIdentifier

	// Return matches for a single tag.
	QueryTagExact(tag string) []ResultIdentifier

	// Return matches that are, or are close to this tag (think of this as 'LIKE %foo%' in SQL).
	QueryTagFuzzy(tag string) []ResultIdentifier

	// Return the results for a typed query.
	// The tag is the type to search for (e.g. 'year'), and the return value is
	// expected to contain a TypedResult with the document identifier and value
	// for the typed query.
	QueryTypedTags(tagType string) []TypedResult

	// Return the cost for querying this exact tag.
	CostTagExact(tag string) int64

	// Return the cost for querying things that are, or are close to, this tag.
	CostTagFuzzy(tag string) int64

	// Return the cost for querying a type.
	CostTypedTags(tagType string) int64

	// Return the cost of querying everything.
	CostAll() int64

	// Filter the current index, using the provided map.
	// Return an index which will return the same data, but only for items
	// inside filteredResults, not the whole index.
	CreateFilteredIndex(filteredResults map[ResultIdentifier]ResultIdentifier) Index
}
