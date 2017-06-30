package generator

import (
	"bytes"
	"io"
	"testing"
)

var golangCommentTests = []struct {
	s, expect       string
	indent, lineLen int
}{
	{
		s:       "This is a comment which should be wrapped.",
		expect:  "\t// This is a comment which\n\t// should be wrapped.\n",
		indent:  1,
		lineLen: 40,
	}, {
		s:       "Areallylongwordcannotbewrapped until after",
		expect:  "// Areallylongwordcannotbewrapped\n// until after\n",
		lineLen: 20,
	}, {
		s:      "Line 1\nLine 2\nLine 3\n",
		expect: "// Line 1\n// Line 2\n// Line 3\n",
	}, {
		s:      "Blank\n\n\nlines",
		expect: "// Blank\n//\n//\n// lines\n",
	}, {
		s:      "List:\n\titem 1\n\titem 2",
		expect: "// List:\n//\titem 1\n//\titem 2\n",
	}, {
		s:       "tabs are\t\t\t\t\twider     than spaces",
		expect:  "// tabs are\n//\t\t\t\t\twider\n//     than spaces\n",
		lineLen: 30,
	},
}

func TestGolangCommentWriter(t *testing.T) {
	for _, test := range golangCommentTests {
		b := new(bytes.Buffer)
		w := GolangCommentWriter(b, test.indent, test.lineLen)

		if _, err := io.WriteString(w, test.s); err != nil {
			t.Errorf("Write() returned error: %s", err.Error())
		}

		if err := w.Close(); err != nil {
			t.Errorf("Close() returned error: %s", err.Error())
		}

		if test.expect != b.String() {
			t.Errorf("expected %q, got %q", test.expect, b.String())
		}
	}
}

func TestRecycledCommentBuffer(t *testing.T) {
	b := new(bytes.Buffer)
	w := GolangCommentWriter(b, 0, 0)

	// write a buffer as usual
	buf := []byte("foo")
	if _, err := w.Write(buf); err != nil {
		t.Errorf("first Write() returned error: %s", err.Error())
	}

	// now recycle that buffer with different data to write
	buf[0] = 'b'
	buf[1] = 'a'
	buf[2] = 'r'
	if _, err := w.Write(buf); err != nil {
		t.Errorf("second Write() returned error: %s", err.Error())
	}

	if err := w.Close(); err != nil {
		t.Errorf("Close() returned error: %s", err.Error())
	}
	expect := "// foobar\n"
	if b.String() != expect {
		t.Errorf("expected %q, got %q", expect, b.String())
	}
}
