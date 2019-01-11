package search

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
)

type searchQuery struct {
	tokens     []string
	currentPos int // in tokens

	queryRoot queryToken
}

type queryToken interface {
	check(index Index) error
	eval(index Index) chan ResultIdentifier
	cost(index Index) int64
}
type notToken struct {
	right queryToken
}

func (this notToken) check(index Index) error {
	return this.right.check(index)
}
func (this notToken) eval(index Index) chan ResultIdentifier {
	resultChan := make(chan ResultIdentifier)

	go func() {
		rhs := this.right.eval(index)
		rhMap := make(map[ResultIdentifier]bool)

		for r := range rhs {
			rhMap[r] = true
		}

		all := index.QueryAll()

		for re := range all {
			if _, ok := rhMap[re]; !ok {
				resultChan <- re
			}
		}

		close(resultChan)
	}()

	return resultChan
}
func (this notToken) cost(index Index) int64 { return index.CostAll() + this.right.cost(index) }

type lParenToken struct{}

func (this lParenToken) check(index Index) error                { panic("unreachable") }
func (this lParenToken) eval(index Index) chan ResultIdentifier { panic("unreachable") }
func (this lParenToken) cost(index Index) int64                 { panic("unreachable") }

type rParenToken struct{}

func (this rParenToken) check(index Index) error                { panic("unreachable") }
func (this rParenToken) eval(index Index) chan ResultIdentifier { panic("unreachable") }
func (this rParenToken) cost(index Index) int64                 { panic("unreachable") }

type virtualToken struct {
	printable string
	realToken queryToken
}

func (this virtualToken) check(index Index) error { return this.realToken.check(index) }
func (this virtualToken) eval(index Index) chan ResultIdentifier {
	return this.realToken.eval(index)
}
func (this virtualToken) cost(index Index) int64 { return this.realToken.cost(index) }

type tagQueryToken struct {
	tag string
}

func (this tagQueryToken) check(index Index) error { return nil }
func (this tagQueryToken) eval(index Index) chan ResultIdentifier {
	return index.QueryTagFuzzy(this.tag)
}
func (this tagQueryToken) cost(index Index) int64 { return index.CostTagFuzzy(this.tag) }

// ### we can't create these at present (there was a way, but it's broken).
// consider how we ought to make this work in parsing.
// perhaps we should repurpose "" to indicate an exact query, and require spaces
// to be \escaped.
type equalsQueryToken struct {
	equals string
}

func (this equalsQueryToken) check(index Index) error { return nil }
func (this equalsQueryToken) eval(index Index) chan ResultIdentifier {
	return index.QueryTagExact(this.equals)
}
func (this equalsQueryToken) cost(index Index) int64 { return index.CostTagExact(this.equals) }

const qDebug = false

// '&&'
type andQueryToken struct {
	left  queryToken
	right queryToken
}

func (this andQueryToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	return rhs
}
func (this andQueryToken) eval(qindex Index) chan ResultIdentifier {
	resultChan := make(chan ResultIdentifier)

	go func() {
		cheapest := this.left
		expensive := this.right

		if this.left.cost(qindex) > this.right.cost(qindex) {
			cheapest = this.right
			expensive = this.left
		}

		log.Printf("Evaluating cheapest: %+v", cheapest)
		cheapResults := cheapest.eval(qindex)
		cheapestMap := make(map[ResultIdentifier]ResultIdentifier)

		for re := range cheapResults {
			cheapestMap[re] = re
		}

		// create a new (filtered) index ...
		nindex := qindex.CreateFilteredIndex(cheapestMap)

		// and use it to evaluate the expensive side (already filtered, by the cheaper side)
		log.Printf("Evaluating expensive side: %+v -- with filtered index (%d filtered)", expensive, len(cheapestMap))
		expensiveResults := expensive.eval(nindex)
		for re := range expensiveResults {
			resultChan <- re
		}
		close(resultChan)
	}()

	return resultChan
}
func (this andQueryToken) cost(index Index) int64 {
	return int64(math.Min(float64(this.left.cost(index)), float64(this.right.cost(index))))
}

// '|&'
type orQueryToken struct {
	left  queryToken
	right queryToken
}

func (this orQueryToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	return rhs
}
func (this orQueryToken) eval(index Index) chan ResultIdentifier {
	resultChan := make(chan ResultIdentifier)

	var wg sync.WaitGroup

	drain := func(qt queryToken) {
		res := qt.eval(index)
		for v := range res {
			resultChan <- v
		}
		wg.Done()
	}

	go func() {
		wg.Add(2)

		go drain(this.left)
		go drain(this.right)

		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}
func (this orQueryToken) cost(index Index) int64 {
	return this.left.cost(index) + this.right.cost(index)
}

// ':'
type colonToken struct{}

func (this colonToken) check(index Index) error                { panic("unreachable") }
func (this colonToken) eval(index Index) chan ResultIdentifier { panic("unreachable") }
func (this colonToken) cost(index Index) int64                 { panic("unreachable") }

// given a query like: 'year:2004', return all entries type-tagged with 'year'.
// this can then be filtered later.
func matchingTagsOfType(index Index, ofType queryToken, tagQueryFunc func(ResultIdentifier, string, string) bool) chan ResultIdentifier {
	resultChan := make(chan ResultIdentifier)

	go func() {
		searchFor := ""
		switch ofTyped := ofType.(type) {
		case tagQueryToken:
			// ### should have a way to query the index for whether or not this
			// is a valid typed search
			searchFor = ofTyped.tag
		}
		log.Printf("Querying for typed tags of type %s", searchFor)

		for typedRe := range index.QueryTypedTags(searchFor) {
			if tagQueryFunc(typedRe.ID, searchFor, typedRe.TypeValue) {
				resultChan <- typedRe.ID
			}
		}

		log.Printf("Done querying for typed tags of type %s", searchFor)
		close(resultChan)
	}()

	return resultChan
}

func numericRightHandSide(rightHand queryToken) (int64, error) {
	rhs := ""
	switch rightTyped := rightHand.(type) {
	case tagQueryToken:
		rhs = rightTyped.tag
	default:
		return 0, (fmt.Errorf("Unexpected right hand side for mathematical comparison: %T", rightTyped))
	}

	var numericValue int64
	_, err := fmt.Sscanf(rhs, "%d", &numericValue)
	if err != nil {
		return 0, (fmt.Errorf("Non-numeric right hand side: %s (%s)", rhs, err))
	}
	return numericValue, nil
}

// '=='
// ### technically, we could/should allow non-numeric equality comparisons for
// typed queries, but so far I have no use for that.
// when we do, consider adding notEqualToToken as well.
type equalToToken struct {
	left  queryToken
	right queryToken
}

func (this equalToToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	if rhs != nil {
		return rhs
	}
	_, err := numericRightHandSide(this.right)
	return err
}

func (this equalToToken) eval(index Index) chan ResultIdentifier {
	wantedVal, _ := numericRightHandSide(this.right)
	return matchingTagsOfType(index, this.left, func(re ResultIdentifier, tag, tagSuffix string) bool {
		val, err := strconv.Atoi(tagSuffix)
		if err != nil {
			log.Printf("Non-numeric tag suffix: %s on tag %s entry %d (%s)", tagSuffix, tag, re, err)
			return false
		}
		if int64(val) == wantedVal {
			return true
		}
		return false
	})
}
func (this equalToToken) cost(index Index) int64 {
	return index.CostTypedTags(this.left.(tagQueryToken).tag) /* ### bad to just cast like this */
}

// '<'
type lessThanToken struct {
	left  queryToken
	right queryToken
}

func (this lessThanToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	if rhs != nil {
		return rhs
	}
	_, err := numericRightHandSide(this.right)
	return err
}
func (this lessThanToken) eval(index Index) chan ResultIdentifier {
	wantedVal, _ := numericRightHandSide(this.right)
	return matchingTagsOfType(index, this.left, func(re ResultIdentifier, tag, tagSuffix string) bool {
		val, err := strconv.Atoi(tagSuffix)
		if err != nil {
			log.Printf("Non-numeric tag suffix: %s on tag %s entry %d (%s)", tagSuffix, tag, re, err)
			return false
		}
		if int64(val) < wantedVal {
			return true
		}
		return false
	})
}
func (this lessThanToken) cost(index Index) int64 {
	return index.CostTypedTags(this.left.(tagQueryToken).tag) /* ### bad to just cast like this */
}

// '>'
type greaterThanToken struct {
	left  queryToken
	right queryToken
}

func (this greaterThanToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	if rhs != nil {
		return rhs
	}
	_, err := numericRightHandSide(this.right)
	return err
}
func (this greaterThanToken) eval(index Index) chan ResultIdentifier {
	wantedVal, _ := numericRightHandSide(this.right)
	return matchingTagsOfType(index, this.left, func(re ResultIdentifier, tag, tagSuffix string) bool {
		val, err := strconv.Atoi(tagSuffix)
		if err != nil {
			log.Printf("Non-numeric tag suffix: %s on tag %s entry %d (%s)", tagSuffix, tag, re, err)
			return false
		}
		if int64(val) > wantedVal {
			return true
		}
		return false
	})
}
func (this greaterThanToken) cost(index Index) int64 {
	return index.CostTypedTags(this.left.(tagQueryToken).tag) /* ### bad to just cast like this */
}

// '<='
type lessThanEqualToken struct {
	left  queryToken
	right queryToken
}

func (this lessThanEqualToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	if rhs != nil {
		return rhs
	}
	_, err := numericRightHandSide(this.right)
	return err
}
func (this lessThanEqualToken) eval(index Index) chan ResultIdentifier {
	wantedVal, _ := numericRightHandSide(this.right)
	return matchingTagsOfType(index, this.left, func(re ResultIdentifier, tag, tagSuffix string) bool {
		val, err := strconv.Atoi(tagSuffix)
		if err != nil {
			log.Printf("Non-numeric tag suffix: %s on tag %s entry %d (%s)", tagSuffix, tag, re, err)
			return false
		}
		if int64(val) <= wantedVal {
			return true
		}
		return false
	})
}
func (this lessThanEqualToken) cost(index Index) int64 {
	return index.CostTypedTags(this.left.(tagQueryToken).tag) /* ### bad to just cast like this */
}

// '>='
type greaterThanEqualToken struct {
	left  queryToken
	right queryToken
}

func (this greaterThanEqualToken) check(index Index) error {
	lhs := this.left.check(index)
	rhs := this.right.check(index)
	if lhs != nil {
		return lhs
	}
	if rhs != nil {
		return rhs
	}
	_, err := numericRightHandSide(this.right)
	return err
}
func (this greaterThanEqualToken) eval(index Index) chan ResultIdentifier {
	wantedVal, _ := numericRightHandSide(this.right)
	return matchingTagsOfType(index, this.left, func(re ResultIdentifier, tag, tagSuffix string) bool {
		val, err := strconv.Atoi(tagSuffix)
		if err != nil {
			log.Printf("Non-numeric tag suffix: %s on tag %s entry %d (%s)", tagSuffix, tag, re, err)
			return false
		}
		if int64(val) >= wantedVal {
			return true
		}
		return false
	})
}
func (this greaterThanEqualToken) cost(index Index) int64 {
	return index.CostTypedTags(this.left.(tagQueryToken).tag) /* ### bad to just cast like this */
}
