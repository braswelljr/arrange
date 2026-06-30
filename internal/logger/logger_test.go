package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		s        string
		maxRunes int
		want     string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello w…"},
		{"", 5, ""},
		{"abc", 0, ""},
		{"ab", 1, "…"},
	}

	for _, tc := range cases {
		got := truncate(tc.s, tc.maxRunes)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.s, tc.maxRunes, got, tc.want)
		}
	}
}

func newLogger() (*Logger, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	return New(out, err), out, err
}

func TestInfo(t *testing.T) {
	l, out, _ := newLogger()
	l.Info("hello info")
	if !strings.Contains(out.String(), "hello info") {
		t.Errorf("Info did not write message to out: %q", out.String())
	}
}

func TestInfof(t *testing.T) {
	l, out, _ := newLogger()
	l.Infof("value is %d", 42)
	if !strings.Contains(out.String(), "value is 42") {
		t.Errorf("Infof did not write formatted message to out: %q", out.String())
	}
}

func TestSuccess(t *testing.T) {
	l, out, _ := newLogger()
	l.Success("done")
	if !strings.Contains(out.String(), "done") {
		t.Errorf("Success did not write message to out: %q", out.String())
	}
}

func TestSuccessf(t *testing.T) {
	l, out, _ := newLogger()
	l.Successf("completed %s", "task")
	if !strings.Contains(out.String(), "completed task") {
		t.Errorf("Successf did not write formatted message to out: %q", out.String())
	}
}

func TestWarn(t *testing.T) {
	l, out, _ := newLogger()
	l.Warn("watch out")
	if !strings.Contains(out.String(), "watch out") {
		t.Errorf("Warn did not write message to out: %q", out.String())
	}
}

func TestWarnf(t *testing.T) {
	l, out, _ := newLogger()
	l.Warnf("threshold %d exceeded", 99)
	if !strings.Contains(out.String(), "threshold 99 exceeded") {
		t.Errorf("Warnf did not write formatted message to out: %q", out.String())
	}
}

func TestError(t *testing.T) {
	l, out, errBuf := newLogger()
	l.Error("something failed")
	if strings.Contains(out.String(), "something failed") {
		t.Errorf("Error wrote to out instead of err")
	}
	if !strings.Contains(errBuf.String(), "something failed") {
		t.Errorf("Error did not write message to err: %q", errBuf.String())
	}
}

func TestErrorf(t *testing.T) {
	l, out, errBuf := newLogger()
	l.Errorf("code %d", 500)
	if strings.Contains(out.String(), "code 500") {
		t.Errorf("Errorf wrote to out instead of err")
	}
	if !strings.Contains(errBuf.String(), "code 500") {
		t.Errorf("Errorf did not write formatted message to err: %q", errBuf.String())
	}
}

func TestEvent(t *testing.T) {
	l, out, _ := newLogger()
	l.Event("file changed")
	if !strings.Contains(out.String(), "file changed") {
		t.Errorf("Event did not write message to out: %q", out.String())
	}
}

func TestEventf(t *testing.T) {
	l, out, _ := newLogger()
	l.Eventf("modified %s", "main.go")
	if !strings.Contains(out.String(), "modified main.go") {
		t.Errorf("Eventf did not write formatted message to out: %q", out.String())
	}
}

func TestMove(t *testing.T) {
	l, out, _ := newLogger()
	l.Move("/src/foo.go", "/dst/bar.go")
	s := out.String()
	if !strings.Contains(s, "foo.go") {
		t.Errorf("Move did not include src in output: %q", s)
	}
	if !strings.Contains(s, "bar.go") {
		t.Errorf("Move did not include dst in output: %q", s)
	}
}

func TestSeparator(t *testing.T) {
	l, out, _ := newLogger()
	l.Separator()
	if !strings.Contains(out.String(), "─") {
		t.Errorf("Separator did not write separator character to out: %q", out.String())
	}
}

func TestHeaderWithVersion(t *testing.T) {
	l, out, _ := newLogger()
	l.Header("myapp", "1.2.3")
	s := out.String()
	if !strings.Contains(s, "myapp") {
		t.Errorf("Header did not write appName to out: %q", s)
	}
	if !strings.Contains(s, "1.2.3") {
		t.Errorf("Header did not write version to out: %q", s)
	}
	if !strings.Contains(s, "─") {
		t.Errorf("Header did not write separator character to out: %q", s)
	}
}

func TestHeaderWithoutVersion(t *testing.T) {
	l, out, _ := newLogger()
	l.Header("myapp", "")
	s := out.String()
	if !strings.Contains(s, "myapp") {
		t.Errorf("Header did not write appName to out: %q", s)
	}
	if strings.Contains(s, "v") {
		t.Errorf("Header included version block when version is empty: %q", s)
	}
}

func TestFooter(t *testing.T) {
	l, out, _ := newLogger()
	l.Footer()
	if !strings.Contains(out.String(), "─") {
		t.Errorf("Footer did not write separator character to out: %q", out.String())
	}
}

func TestPrintln(t *testing.T) {
	l, out, _ := newLogger()
	l.Println("some message")
	if !strings.Contains(out.String(), "some message") {
		t.Errorf("Println did not write message to out: %q", out.String())
	}
}

func TestPrintf(t *testing.T) {
	l, out, _ := newLogger()
	l.Printf("count: %d\n", 7)
	if !strings.Contains(out.String(), "count: 7") {
		t.Errorf("Printf did not write formatted message to out: %q", out.String())
	}
}
