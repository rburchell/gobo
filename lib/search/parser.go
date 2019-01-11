package search

import (
	"fmt"
	"log"
)

// Turn a string into a series of tokens.
// For us, this means: "year<2004" => "year" "<" "2004"
func tokenize(query string) ([]string, error) {
	tokens := []string{}
	stringTok := ""

	isSpecialToken := func(b byte) bool {
		switch b {
		case '!':
			return true
		case '=':
			return true
		case '>':
			return true
		case '<':
			return true
		case '(':
			return true
		case ')':
			return true
		case '&':
			return true
		case '|':
			return true
		case ':':
			return true
		}
		return false
	}

	appendIfMathToken := func() {
		if len(stringTok) == 1 {
			if isSpecialToken(stringTok[0]) {
				tokens = append(tokens, stringTok)
				stringTok = ""
			}
		}
	}
	appendToken := func() {
		if len(stringTok) != 0 {
			tokens = append(tokens, stringTok)
			stringTok = ""
		}
	}

	for pos := 0; pos < len(query); pos++ {
		c := query[pos]
		switch {
		case isSpecialToken(c):
			appendToken()
			stringTok = string(c)
		case c == ' ':
			appendToken()
		case c == '"':
			foundEnd := false

			pos++ // skip first "

			for pos < len(query) {
				cchar := query[pos]
				pos++
				if cchar == '"' {
					foundEnd = true
					break
				} else {
					stringTok += string(cchar)
				}
			}

			pos-- // rewind one for the loop
			if !foundEnd {
				return nil, fmt.Errorf("Unterminated string literal \"")
			}
			appendToken()
		default:
			// '<2004' should be two tokens, not one.
			appendIfMathToken()
			stringTok += string(c)
		}
	}

	appendToken() // append any leftover

	//log.Printf("After tokenize")
	//for i,t := range tokens {
	//	log.Printf("tok %d: %s", i,t)
	//}
	return tokens, nil
}

// A TokenReplacement can be thought of as a search and replace on the search query.
// For instance, if you want to rewrite queries like "with:friends" to
// "alice && bob", you could do that here. Just set:
//
// Search: []string{"with", ":", "friends"}
// Replace: []string{"alice", "&&", "bob"}
type TokenReplacement struct {
	Search  []string
	Replace []string
}

// Replace a series of tokens into another series of tokens.
//
// This will happen iteratively, until there is nothing left to replace.
// For example, "|" "|" => "||"
func (this *searchQuery) Replace(replacements []TokenReplacement) {
	replaced := []string{}
	replaceAgain := true
	for replaceAgain {
		for idx := 0; idx < len(this.tokens); idx++ {
			tok := this.tokens[idx]
			wasReplaced := false

			for _, rep := range replacements {
				found := true
				for sidx := 0; sidx < len(rep.Search); sidx++ {
					if idx+sidx >= len(this.tokens) {
						found = false
						break
					}

					if this.tokens[idx+sidx] != rep.Search[sidx] {
						found = false
						break
					}
				}

				if found {
					replaced = append(replaced, rep.Replace...)
					replaced = append(replaced, this.tokens[idx+len(rep.Search):]...)
					// start again
					this.tokens = this.tokens[0:0]
					for _, tok := range replaced {
						this.tokens = append(this.tokens, tok)
					}
					replaced = replaced[0:0]
					idx = -1
					wasReplaced = true
					break
				}
			}

			if !wasReplaced {
				replaced = append(replaced, tok)

				if idx == len(this.tokens)-1 {
					replaceAgain = false
					break
				}
			}
		}
	}

	//log.Printf("After replace")
	//for i,t := range replaced {
	//	log.Printf("tok %d: %s", i,t)
	//}
	this.tokens = replaced
}

func (this *searchQuery) peekToken() queryToken {
	lastPos := this.currentPos
	tok := this.readToken()
	this.currentPos = lastPos
	return tok
}

func (this *searchQuery) readToken() queryToken {
	if this.currentPos == len(this.tokens) {
		return nil
	}
	var ret queryToken
	switch this.tokens[this.currentPos] {
	case "&&":
		ret = andQueryToken{}
	case "||":
		ret = orQueryToken{}
	case "!":
		ret = notToken{}
	case "(":
		ret = lParenToken{}
	case ")":
		ret = rParenToken{}
	case ">":
		ret = greaterThanToken{}
	case ">=":
		ret = greaterThanEqualToken{}
	case "<":
		ret = lessThanToken{}
	case "<=":
		ret = lessThanEqualToken{}
	case "==":
		ret = equalToToken{}
	case ":":
		ret = colonToken{}
	default:
		ret = tagQueryToken{tag: this.tokens[this.currentPos]}
	}
	this.currentPos++
	return ret
}

// 'greedy': whether or not to read more than a single token
// basically a cheap way of making 'foo && bar<5' parse the same as 'foo && (bar<5)'.
func (this *searchQuery) parse(greedy bool) (queryToken, error) {
	left := this.readToken()
	poss := this.peekToken()

	var err error

	switch tleft := left.(type) {
	case greaterThanToken:
		return nil, fmt.Errorf("Unexpected >")
	case lessThanToken:
		return nil, fmt.Errorf("Unexpected <")
	case andQueryToken:
		return nil, fmt.Errorf("Unexpected &&")
	case orQueryToken:
		return nil, fmt.Errorf("Unexpected &&")
	case rParenToken:
		return nil, fmt.Errorf("Unexpected )")
	case lParenToken:
		// we ate the ( already
		left, err = this.parse(true) // fetch the expr
		if err != nil {
			return nil, err
		}
		rparen := this.readToken() // eat the )
		if _, ok := rparen.(rParenToken); !ok {
			return nil, fmt.Errorf("Expected: ), got %+v", poss)
		}
		// handle binary conditions below
	case notToken:
		if poss == nil {
			return nil, fmt.Errorf("Expected expression for !")
		}
		tleft.right, err = this.parse(true)
		if err != nil {
			return nil, err
		}
		return tleft, nil
	case tagQueryToken:
		if poss != nil {
			if _, ok := poss.(colonToken); ok {
				this.readToken() // consume :
				poss = this.peekToken()
				if secondPart, ok := poss.(tagQueryToken); !ok {
					return nil, fmt.Errorf("Expected: tag after colon")
				} else {
					left = virtualToken{printable: fmt.Sprintf("%s:%s", tleft.tag, secondPart.tag), realToken: equalsQueryToken{equals: fmt.Sprintf("%s:%s", tleft.tag, secondPart.tag)}}
					this.readToken() // consume secondPart
				}
			}
		}
	}

	poss = this.peekToken() // update peeked token

	// handle left/right hand side operators
	if greedy && poss != nil {
		var right queryToken
		switch poss.(type) {
		case equalToToken:
			this.readToken()
			right, err = this.parse(false)
			if err != nil {
				return nil, err
			}
			left = equalToToken{left: left, right: right}
		case lessThanEqualToken:
			this.readToken()
			right, err = this.parse(false)
			if err != nil {
				return nil, err
			}
			left = lessThanEqualToken{left: left, right: right}
		case lessThanToken:
			this.readToken()
			right, err = this.parse(false)
			if err != nil {
				return nil, err
			}
			left = lessThanToken{left: left, right: right}
		case greaterThanEqualToken:
			this.readToken()
			right, err = this.parse(false)
			if err != nil {
				return nil, err
			}
			left = greaterThanEqualToken{left: left, right: right}
		case greaterThanToken:
			this.readToken()
			right, err = this.parse(false)
			if err != nil {
				return nil, err
			}
			left = greaterThanToken{left: left, right: right}
		}
	}

	// now handle boolean operators
	// this has to be separate from the above, so: year<2005 && foo parses as (year<2005) && foo.
	// with them separate, 'left' will now be lessThanEqualToken, meaning we're ready to rock.
	poss = this.peekToken()

	if greedy && poss != nil {
		switch tposs := poss.(type) {
		case andQueryToken:
			tposs.left = left
			this.readToken() // ignore &&
			tposs.right, err = this.parse(true)
			if err != nil {
				return nil, err
			}
			return tposs, nil
		case orQueryToken:
			tposs.left = left
			this.readToken() // ignore ||
			tposs.right, err = this.parse(true)
			if err != nil {
				return nil, err
			}
			return tposs, nil
		case rParenToken:
			// ok; ignore (at this point, we must be the end of an lParenToken).
		default:
			return nil, fmt.Errorf("Unexpected token %+v (left hand side: %+v)", poss, left)
		}
	}

	return left, nil
}

// ### member function on searchQuery might make sense
func Evaluate(searchQuery *searchQuery, index Index) (chan ResultIdentifier, error) {
	var err error
	searchQuery.queryRoot, err = searchQuery.parse(true)
	if err != nil {
		return nil, fmt.Errorf("Error parsing: %s", err)
	}
	err = searchQuery.queryRoot.check(index)
	if err != nil {
		return nil, fmt.Errorf("Query is invalid: %s", err)
	}

	// ### left in for debug, but take out later
	printQuery(searchQuery, index)
	return searchQuery.queryRoot.eval(index), nil
}

func printQuery(searchQuery *searchQuery, index Index) {
	printTokenTree(searchQuery.queryRoot, 0, index)
}

func printTokenTree(node queryToken, indentLevel int, index Index) {
	indentStr := ""
	indentLevel += 1
	for i := 0; i < indentLevel; i++ {
		indentStr += "\t"
	}

	switch tn := node.(type) {
	case andQueryToken:
		log.Printf("%sand: (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case orQueryToken:
		log.Printf("%sor: (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case tagQueryToken:
		log.Printf("%stag ~= %s (cost %d)", indentStr, tn.tag, tn.cost(index))
	case equalsQueryToken:
		log.Printf("%stag == %s (cost %d)", indentStr, tn.equals, tn.cost(index))
	case equalToToken:
		log.Printf("%s== (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case greaterThanToken:
		log.Printf("%s>: (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case lessThanToken:
		log.Printf("%s< (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case greaterThanEqualToken:
		log.Printf("%s>= (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case lessThanEqualToken:
		log.Printf("%s<= (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.left, indentLevel, index)
		printTokenTree(tn.right, indentLevel, index)
	case notToken:
		log.Printf("%snot: (cost %d)", indentStr, tn.cost(index))
		printTokenTree(tn.right, indentLevel, index)
	case virtualToken:
		log.Printf("%s%s (cost %d)", indentStr, tn.printable, tn.cost(index))
	default:
		log.Printf("%sunknown token type %T", indentStr, node)
	}
}

// Create a query.
// The query can subsequently be altered (using Replace()), or executed using Evaluate.
func CreateQuery(query string) (*searchQuery, error) {
	q := &searchQuery{}
	var err error
	q.tokens, err = tokenize(query)
	if err != nil {
		return nil, fmt.Errorf("Error tokenizing: %s", err)
	}

	// we do these replacements internally, because we want to provide a
	// consistent interface. additional replacements are left up to the package
	// user.
	q.Replace([]TokenReplacement{
		TokenReplacement{
			Search:  []string{"<", "="},
			Replace: []string{"<="},
		},
		TokenReplacement{
			Search:  []string{">", "="},
			Replace: []string{">="},
		},
		TokenReplacement{
			Search:  []string{"=", "="},
			Replace: []string{"=="},
		},
		TokenReplacement{
			Search:  []string{"&", "&"},
			Replace: []string{"&&"},
		},
		TokenReplacement{
			Search:  []string{"|", "|"},
			Replace: []string{"||"},
		},
	})
	return q, nil
}
