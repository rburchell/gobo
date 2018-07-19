package desktop_parser

import (
	"fmt"
)

type state uint8

// The parser works as a simple state machine.
// These are the states, and roughly what they are for.
const (
	// The entry state. Only valid transition is to state_section, or state_key
	// once a section has been identified.
	state_none state = iota

	// Transitions back to state_none when a section is found.
	state_section

	// Transitions to state_key_locale when a [ is found in a key.
	state_key

	// Transitions to state_key_locale_post when a ] is found after a locale.
	state_key_locale

	// Only allow equals or whitespace. Transitions to state_value_pre.
	state_key_locale_post

	// Eats whitespace. Transitions to state_value on anything else.
	state_value_pre

	// Reads a value. Transitions back to state_none once done.
	state_value
)

type parser struct {
	sections []DesktopSection

	// Current section name being read.
	sectionName string

	// Current key name being read.
	keyName string

	// Current key locale being read (if any).
	keyLocale string

	// Current value being read.
	value string
}

func (this *parser) parseStateNone(c rune) (state, error) {
	switch c {
	case '[': // [Desktop Entry
		this.sectionName = ""
		return state_section, nil
	case '\n':
		return state_none, nil
	default:
		if this.sectionName != "" {
			return this.parseStateKey(c)
		} else {
			return state_none, fmt.Errorf("Unexpected character outside a section: %c", c)
		}
	}
}

func (this *parser) parseStateSection(c rune) (state, error) {
	switch c {
	case ']': // Desktop Entry]
		this.sections = append(this.sections, DesktopSection{Name: this.sectionName})
		return state_none, nil
	default:
		this.sectionName += string(c)
	}

	return state_section, nil
}

func (this *parser) parseStateKey(c rune) (state, error) {
	switch {
	case c == '[':
		if this.keyLocale != "" {
			return state_none, fmt.Errorf("Already found a language code: %s", this.keyLocale)
		}
		return state_key_locale, nil
	case c == '=':
		if this.keyName == "" {
			return state_none, fmt.Errorf("Empty key found")
		}
		return state_value_pre, nil
	case c == ' ':
		// ignore
	case c == '-':
		fallthrough
	case c >= '0' && c <= '9':
		fallthrough
	case c >= 'a' && c <= 'z':
		fallthrough
	case c >= 'A' && c <= 'Z':
		this.keyName += string(c)
	default:
		return state_none, fmt.Errorf("Bad key character: %c", c)
	}

	return state_key, nil
}

func (this *parser) parseStateKeyLocale(c rune) (state, error) {
	switch {
	case c == ']':
		return state_key, nil
	default:
		this.keyLocale += string(c)
	}

	return state_key_locale, nil
}

func (this *parser) parseStateKeyLocalePost(c rune) (state, error) {
	switch {
	case c == ' ':
		return state_key_locale_post, nil
	case c == '=':
		return state_value_pre, nil
	default:
		return state_none, fmt.Errorf("Unexpected character after language code %s: %c", this.keyLocale, c)
	}

	return state_key_locale_post, nil
}

func (this *parser) parseStateValuePre(c rune) (state, error) {
	switch c {
	case ' ': // skip spaces
		return state_value_pre, nil
	default:
		return state_value, nil
	}
}

func (this *parser) endValue() {
	if this.keyName == "" {
		return
	}

	i := len(this.sections) - 1
	this.sections[i].Values = append(this.sections[i].Values, DesktopValue{Key: this.keyName, Locale: this.keyLocale, Value: this.value})
	this.keyName = ""
	this.value = ""
	this.keyLocale = ""
}

func (this *parser) parseStateValue(c rune) (state, error) {
	switch c {
	case '\n':
		this.endValue()
		return state_none, nil
	default:
		this.value += string(c)
	}

	return state_value, nil
}
