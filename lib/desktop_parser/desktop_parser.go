package desktop_parser

import (
	"bufio"
	"fmt"
	"io"
)

type state uint8

const (
	state_none state = iota
	state_section
	state_key
	state_key_locale      // optional, Name[en]
	state_key_locale_post // only allow equals or whitespace.
	state_value_pre
	state_value
)

type parser struct {
	sections    []DesktopSection
	sectionName string
	keyName     string
	keyLocale   string
	value       string
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

type DesktopFile struct {
	Sections []DesktopSection
}

func (this *DesktopFile) FindFirst(key string) *DesktopValue {
	return this.Sections[0].FindFirst(key)
}

func (this *DesktopFile) FindAll(key string) []DesktopValue {
	return this.Sections[0].FindAll(key)
}

type DesktopSection struct {
	Name   string
	Values []DesktopValue
}

func (this DesktopSection) FindFirst(key string) *DesktopValue {
	for _, v := range this.Values {
		if v.Key == key {
			return &v
		}
	}

	return nil
}

func (this DesktopSection) FindAll(key string) []DesktopValue {
	ret := []DesktopValue{}
	for _, v := range this.Values {
		if v.Key == key {
			ret = append(ret, v)
		}
	}

	return ret
}

type DesktopValue struct {
	Key    string
	Locale string
	Value  string
}

func Parse(fd io.Reader) (*DesktopFile, error) {
	lineState := state_none

	l := parser{}
	br := bufio.NewReader(fd)

	for {
		c, _, err := br.ReadRune()
		if err != nil {
			if err == io.EOF {
				l.endValue()

				// ### should do some cleanup/validation
				// ensure there's a Desktop Entry section
				// set it as the default on the Desktop File rather than assuming it's first
				// don't allow duplicate sections
				// don't allow duplicate key+locale in a section

				return &DesktopFile{l.sections}, nil
			} else {
				return nil, fmt.Errorf("Error while reading: %s", err)
			}
		}

		switch lineState {
		case state_none:
			lineState, err = l.parseStateNone(c)
			if err != nil {
				return nil, err
			}
		case state_section:
			lineState, err = l.parseStateSection(c)
			if err != nil {
				return nil, err
			}
		case state_key:
			lineState, err = l.parseStateKey(c)
			if err != nil {
				return nil, err
			}
		case state_key_locale:
			lineState, err = l.parseStateKeyLocale(c)
			if err != nil {
				return nil, err
			}
		case state_key_locale_post:
			lineState, err = l.parseStateKeyLocalePost(c)
			if err != nil {
				return nil, err
			}
		case state_value_pre:
			lineState, err = l.parseStateValuePre(c)
			if err != nil {
				return nil, err
			}

			if lineState == state_value {
				// whitespace all skipped. read a value char.
				lineState, err = l.parseStateValue(c)
				if err != nil {
					return nil, err
				}
			}
		case state_value:
			lineState, err = l.parseStateValue(c)
			if err != nil {
				return nil, err
			}
		default:
			panic(fmt.Sprintf("Unknown state %d", lineState))
		}
	}
}
