package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const fixtureHTML = `
<html>
  <body>
    <p id="p2" class="sb">
      <span id="v1-1-1-1" class="v">
        <a class="cl vx vp"><strong>1</strong> </a>En el principio, Dios creó los cielos y la tierra.<a class="b">+</a>
      </span>
    </p>
    <p id="p3" class="sb">
      <span id="v1-1-2-1" class="v">
        <a class="vl vx vp">2&#8239;</a>Ahora bien, la tierra no tenía forma y estaba vacía.<a class="fn">*</a> La oscuridad cubría.<a class="fn pr">*</a><a class="b">+</a>
      </span>
      <span id="v1-1-3-1" class="v">
        <a class="vl vx vp">3&#8239;</a>Y Dios dijo: “Que haya luz”. Así que hubo luz.<a class="b">+</a>
      </span>
    </p>
  </body>
</html>`

func TestParseCitation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		bookID    int
		chapter   int
		verses    []int
		multiline bool
		reference string
	}{
		{
			name:      "single",
			input:     "Genesis 1:1",
			bookID:    1,
			chapter:   1,
			verses:    []int{1},
			reference: "Génesis 1:1",
		},
		{
			name:      "list",
			input:     "Genesis 1:1, 2, 5",
			bookID:    1,
			chapter:   1,
			verses:    []int{1, 2, 5},
			multiline: true,
			reference: "Génesis 1:1, 2, 5",
		},
		{
			name:      "range",
			input:     "Genesis 1:1-3",
			bookID:    1,
			chapter:   1,
			verses:    []int{1, 2, 3},
			reference: "Génesis 1:1-3",
		},
		{
			name:      "short gen",
			input:     "Gen 1:1",
			bookID:    1,
			chapter:   1,
			verses:    []int{1},
			reference: "Génesis 1:1",
		},
		{
			name:      "short ge",
			input:     "Ge 1:1",
			bookID:    1,
			chapter:   1,
			verses:    []int{1},
			reference: "Génesis 1:1",
		},
		{
			name:      "accented",
			input:     "Génesis 1:1",
			bookID:    1,
			chapter:   1,
			verses:    []int{1},
			reference: "Génesis 1:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCitation(tt.input)
			if err != nil {
				t.Fatalf("parseCitation() error = %v", err)
			}
			if got.BookID != tt.bookID {
				t.Fatalf("BookID = %d, want %d", got.BookID, tt.bookID)
			}
			if got.Chapter != tt.chapter {
				t.Fatalf("Chapter = %d, want %d", got.Chapter, tt.chapter)
			}
			if !reflect.DeepEqual(got.VerseNumbers(), tt.verses) {
				t.Fatalf("VerseNumbers() = %v, want %v", got.VerseNumbers(), tt.verses)
			}
			if got.Multiline != tt.multiline {
				t.Fatalf("Multiline = %v, want %v", got.Multiline, tt.multiline)
			}
			if got.Reference() != tt.reference {
				t.Fatalf("Reference() = %q, want %q", got.Reference(), tt.reference)
			}
		})
	}
}

func TestParseCitationErrors(t *testing.T) {
	tests := []string{
		"Unknown 1:1",
		"Genesis 1",
		"Genesis 1:3-1",
		"Genesis 1:abc",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := parseCitation(input); err == nil {
				t.Fatalf("parseCitation(%q) succeeded, want error", input)
			}
		})
	}
}

func TestExtractVerses(t *testing.T) {
	verses, err := extractVerses([]byte(fixtureHTML), 1, 1, []int{1, 2, 3})
	if err != nil {
		t.Fatalf("extractVerses() error = %v", err)
	}

	want := []Verse{
		{Number: 1, Text: "1 En el principio, Dios creó los cielos y la tierra."},
		{Number: 2, Text: "2 Ahora bien, la tierra no tenía forma y estaba vacía. La oscuridad cubría."},
		{Number: 3, Text: "3 Y Dios dijo: “Que haya luz”. Así que hubo luz."},
	}
	if !reflect.DeepEqual(verses, want) {
		t.Fatalf("extractVerses() = %#v, want %#v", verses, want)
	}

	for _, verse := range verses {
		if strings.ContainsAny(verse.Text, "*+") {
			t.Fatalf("verse text still contains note/reference marker: %q", verse.Text)
		}
	}
}

func TestRunWithArgsRange(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"Genesis", "1:1-2"}, strings.NewReader(""), &stdout, &stderr, testFetcher(t))
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}

	want := "1 En el principio, Dios creó los cielos y la tierra. 2 Ahora bien, la tierra no tenía forma y estaba vacía. La oscuridad cubría.\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunWithArgsList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"Genesis", "1:1,", "2"}, strings.NewReader(""), &stdout, &stderr, testFetcher(t))
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}

	want := "1 En el principio, Dios creó los cielos y la tierra.\n2 Ahora bien, la tierra no tenía forma y estaba vacía. La oscuridad cubría.\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunWithStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(nil, strings.NewReader("Gen 1:1\n"), &stdout, &stderr, testFetcher(t))
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}

	want := "1 En el principio, Dios creó los cielos y la tierra.\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--json", "Ge", "1:1"}, strings.NewReader(""), &stdout, &stderr, testFetcher(t))
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}

	var result Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; stdout = %s", err, stdout.String())
	}
	if result.Reference != "Génesis 1:1" {
		t.Fatalf("Reference = %q, want %q", result.Reference, "Génesis 1:1")
	}
	if result.SourceURL != "https://example.test/1/1" {
		t.Fatalf("SourceURL = %q, want fixture URL", result.SourceURL)
	}
	if len(result.Verses) != 1 || result.Verses[0].Number != 1 {
		t.Fatalf("Verses = %#v, want one verse numbered 1", result.Verses)
	}
	if result.Text != "1 En el principio, Dios creó los cielos y la tierra." {
		t.Fatalf("Text = %q", result.Text)
	}
}

func TestRunFetcherError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	fetch := func(context.Context, int, int) ([]byte, string, error) {
		return nil, "", errors.New("network failed")
	}

	code := run([]string{"Genesis", "1:1"}, strings.NewReader(""), &stdout, &stderr, fetch)
	if code == 0 {
		t.Fatalf("run() code = 0, want failure")
	}
	if !strings.Contains(stderr.String(), "network failed") {
		t.Fatalf("stderr = %q, want network error", stderr.String())
	}
}

func testFetcher(t *testing.T) chapterFetcher {
	t.Helper()
	return func(ctx context.Context, bookID int, chapter int) ([]byte, string, error) {
		if ctx == nil {
			t.Fatal("context is nil")
		}
		if bookID != 1 || chapter != 1 {
			return nil, "", fmt.Errorf("unexpected chapter request: book=%d chapter=%d", bookID, chapter)
		}
		return []byte(fixtureHTML), "https://example.test/1/1", nil
	}
}
