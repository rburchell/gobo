package desktop_parser

import (
	"bytes"
	"fmt"
	"github.com/stvp/assert"
	"testing"
)

type desktopTest struct {
	input         string
	expectedError error
	output        *DesktopFile
}

func TestSimple(t *testing.T) {
	tests := []desktopTest{
		desktopTest{
			input:         "a",
			output:        nil,
			expectedError: fmt.Errorf("Unexpected character outside a section: a"),
		},
		desktopTest{
			input:  "",
			output: &DesktopFile{},
		},
		desktopTest{
			input: "[Desktop Entry]",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
					},
				},
			},
		},
		desktopTest{
			input: "[Desktop Entry]\nA=B",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
						Values: []DesktopValue{
							DesktopValue{
								Key:   "A",
								Value: "B",
							},
						},
					},
				},
			},
		},
		desktopTest{
			input: "[Desktop Entry]\nA=B\nC=D",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
						Values: []DesktopValue{
							DesktopValue{
								Key:   "A",
								Value: "B",
							},
							DesktopValue{
								Key:   "C",
								Value: "D",
							},
						},
					},
				},
			},
		},
		desktopTest{
			input: "[Desktop Entry]\n[Another Section]",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
					},
					DesktopSection{
						Name: "Another Section",
					},
				},
			},
		},
		desktopTest{
			input: "[Desktop Entry]\nA=B\nC=D\n[More values]\n1=2\n3=4",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
						Values: []DesktopValue{
							DesktopValue{
								Key:   "A",
								Value: "B",
							},
							DesktopValue{
								Key:   "C",
								Value: "D",
							},
						},
					},
					DesktopSection{
						Name: "More values",
						Values: []DesktopValue{
							DesktopValue{
								Key:   "1",
								Value: "2",
							},
							DesktopValue{
								Key:   "3",
								Value: "4",
							},
						},
					},
				},
			},
		},
		desktopTest{
			input: "[Desktop Entry]\nA[en]=B\nA[no]=D",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
						Values: []DesktopValue{
							DesktopValue{
								Key:    "A",
								Locale: "en",
								Value:  "B",
							},
							DesktopValue{
								Key:    "A",
								Locale: "no",
								Value:  "D",
							},
						},
					},
				},
			},
		},
		desktopTest{
			input: "[Desktop Entry]\nX-Something-Special=A",
			output: &DesktopFile{
				Sections: []DesktopSection{
					DesktopSection{
						Name: "Desktop Entry",
						Values: []DesktopValue{
							DesktopValue{
								Key:   "X-Something-Special",
								Value: "A",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		b := bytes.NewBufferString(tc.input)
		df, err := Parse(b)
		assert.Equal(t, err, tc.expectedError)
		assert.Equal(t, df, tc.output)
		//log.Printf("Passed: %s == %+v", tc.input, df)
	}
}
