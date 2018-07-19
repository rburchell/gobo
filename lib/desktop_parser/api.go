package desktop_parser

import (
	"bufio"
	"fmt"
	"io"
)

// A desktop file . Consists of multiple sections, which then consist of values.
type DesktopFile struct {
	// All the sections in this file.
	Sections []DesktopSection
}

// Find the first value matching a key in this desktop file.
// This searches the "primary" section only.
func (this *DesktopFile) FindFirst(key string) *DesktopValue {
	return this.Sections[0].FindFirst(key)
}

// Find all values matching a key in this desktop file.
// This searches the "primary" section only.
func (this *DesktopFile) FindAll(key string) []DesktopValue {
	return this.Sections[0].FindAll(key)
}

// Represents a section in the desktop file.
type DesktopSection struct {
	// e.g. "Desktop Entry"
	Name string

	// The key:value pairs in this section.
	Values []DesktopValue
}

// Find a value with a given key in this section.
func (this DesktopSection) FindFirst(key string) *DesktopValue {
	for _, v := range this.Values {
		if v.Key == key {
			return &v
		}
	}

	return nil
}

// Find all values with a given key in this section.
func (this DesktopSection) FindAll(key string) []DesktopValue {
	ret := []DesktopValue{}
	for _, v := range this.Values {
		if v.Key == key {
			ret = append(ret, v)
		}
	}

	return ret
}

// A desktop value is a key:value pair in a desktop file. It may also contain a
// locale, which is considered to be part of the key.
type DesktopValue struct {
	// e.g. Name
	Key string

	// e.g. en
	Locale string

	// everything after the equals.
	Value string
}

// Create a DesktopFile by reading from a fd.
// Returns a pointer to the DesktopFile, or nil (and an error).
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
