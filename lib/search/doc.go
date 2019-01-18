// The search package implements a simple query parser and evaluator for
// (ideally) efficient search on a document store of tagged documents.
//
// The basic idea is that you have a set of (tagged) documents, in an Index,
// let's say they're your family photo collection.
//
// You want to write a simple query like: "year:2011 && in:europe && !germany"
// to find all your photos from that time -- this package will help you do that.
//
// In order to do that, you need to write a concrete implementation of the Index
// interface. It contains a number of functions that the engine will use to
// build the results of a query, like: "give me all tags like this one".
//
// The engine will first parse a query, then ask the index for the cost to
// evaluate each step of the query (this doesn't need to be exact, it's just to
// help the engine evaluate faster), and will then try to evaluate the query in
// the most performant way possible (by using smaller cost nodes before larger
// cost nodes, in order to quickly filter down results to the smallest possible
// subset of results).
//
// The query language has a few different ideas behind it.
// * tags (e.g. 'germany'). These are searched fuzzily, unless queried exactly
//   (### they can't be right now)
// * typed tags (e.g. year:2010). Generally, with a large collection of
//   documents, you will have specific pieces of metadata you can provide for them
//   in a key:value form. A typed tag is essentially key:value. They can be
//   queried exactly (e.g. asking for year:2010), or by range (e.g. year<2010).
//
//   Right now, only numeric typed tags are supported, but in future this may change.
//
//   For typed tags, you can query numerically: >, >=, <, <=, ==.
// * the usual booleans: &&, ||, !
package search
