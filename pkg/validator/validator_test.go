package validator

import (
	"bytes"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
)

// fakeFile adapts a *bytes.Reader to the multipart.File interface
// (io.Reader + io.ReaderAt + io.Seeker + io.Closer).
type fakeFile struct {
	*bytes.Reader
}

func (f fakeFile) Close() error { return nil }

var _ multipart.File = fakeFile{}

func newFile(content []byte) multipart.File {
	return fakeFile{bytes.NewReader(content)}
}

// pad returns prefix followed by enough filler to reach n bytes so that the
// header.Size >= 1024 minimum in ValidateVideoFile is satisfied by real content.
func pad(prefix []byte, n int) []byte {
	out := make([]byte, n)
	copy(out, prefix)
	return out
}

// ISO-BMFF header: 4-byte big-endian box size, then "ftyp", then a brand.
func ftypHeader(boxSize []byte, brand string) []byte {
	h := make([]byte, 0, 12)
	h = append(h, boxSize...)
	h = append(h, 'f', 't', 'y', 'p')
	h = append(h, []byte(brand)...)
	return h
}

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		name string
		buf  []byte
		want bool
	}{
		{
			// Regression: the old implementation hardcoded box size 0x00000018,
			// so every mp4 with any other legal box size was rejected.
			name: "mp4 with box size 0x20 (non-0x18)",
			buf:  ftypHeader([]byte{0x00, 0x00, 0x00, 0x20}, "isom"),
			want: true,
		},
		{
			name: "mp4 with box size 0x18",
			buf:  ftypHeader([]byte{0x00, 0x00, 0x00, 0x18}, "mp42"),
			want: true,
		},
		{
			name: "mp4 with box size 0x1c",
			buf:  ftypHeader([]byte{0x00, 0x00, 0x00, 0x1C}, "isom"),
			want: true,
		},
		{
			name: "mov (ISO-BMFF, qt brand)",
			buf:  ftypHeader([]byte{0x00, 0x00, 0x00, 0x14}, "qt  "),
			want: true,
		},
		{
			name: "mkv/webm EBML",
			buf:  []byte{0x1A, 0x45, 0xDF, 0xA3, 0x01, 0x02, 0x03, 0x04},
			want: true,
		},
		{
			name: "EBML exactly 4 bytes",
			buf:  []byte{0x1A, 0x45, 0xDF, 0xA3},
			want: true,
		},
		{
			name: "avi RIFF....AVI ",
			buf:  []byte{'R', 'I', 'F', 'F', 0x24, 0x10, 0x00, 0x00, 'A', 'V', 'I', ' '},
			want: true,
		},
		{
			name: "RIFF but WAVE form type",
			buf:  []byte{'R', 'I', 'F', 'F', 0x24, 0x10, 0x00, 0x00, 'W', 'A', 'V', 'E'},
			want: false,
		},
		{
			name: "RIFF truncated before form type",
			buf:  []byte{'R', 'I', 'F', 'F', 0x24, 0x10, 0x00, 0x00},
			want: false,
		},
		{
			name: "random bytes",
			buf:  []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x11, 0x22, 0x33, 0x44},
			want: false,
		},
		{
			name: "plain text",
			buf:  []byte("this is definitely not a video file at all"),
			want: false,
		},
		{
			name: "png",
			buf:  []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
			want: false,
		},
		{
			name: "empty buffer must not panic",
			buf:  []byte{},
			want: false,
		},
		{
			name: "nil buffer must not panic",
			buf:  nil,
			want: false,
		},
		{
			name: "buffer shorter than 8 bytes must not panic",
			buf:  []byte{0x00, 0x00, 0x00, 0x20, 'f', 't', 'y'},
			want: false,
		},
		{
			name: "single byte must not panic",
			buf:  []byte{0x1A},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVideoFile(tt.buf); got != tt.want {
				t.Errorf("isVideoFile(%v) = %v, want %v", tt.buf, got, tt.want)
			}
		})
	}
}

func TestValidateVideoFile(t *testing.T) {
	const maxSize = 10 * 1024 * 1024

	mp4 := pad(ftypHeader([]byte{0x00, 0x00, 0x00, 0x20}, "isom"), 2048)
	mov := pad(ftypHeader([]byte{0x00, 0x00, 0x00, 0x14}, "qt  "), 2048)
	mkv := pad([]byte{0x1A, 0x45, 0xDF, 0xA3}, 2048)
	avi := pad([]byte{'R', 'I', 'F', 'F', 0x24, 0x10, 0x00, 0x00, 'A', 'V', 'I', ' '}, 2048)
	junk := pad([]byte("not a video, just some bytes"), 2048)

	tests := []struct {
		name     string
		filename string
		content  []byte
		size     int64 // 0 means: use len(content)
		maxSize  int64
		wantErr  error // nil means no error expected
	}{
		{name: "mp4 non-0x18 box size accepted", filename: "clip.mp4", content: mp4},
		{name: "mov accepted", filename: "clip.mov", content: mov},
		{name: "mkv accepted", filename: "clip.mkv", content: mkv},
		{name: "webm accepted", filename: "clip.webm", content: mkv},
		{name: "avi accepted", filename: "clip.avi", content: avi},
		{name: "uppercase extension accepted", filename: "CLIP.MP4", content: mp4},
		{
			name:     "random bytes rejected",
			filename: "clip.mp4",
			content:  junk,
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "content shorter than 8 bytes rejected without panic",
			filename: "clip.mp4",
			content:  []byte{0x00, 0x00, 0x00},
			size:     2048, // claim a valid size so we reach the sniffing step
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "disallowed extension rejected",
			filename: "clip.exe",
			content:  mp4,
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "no extension rejected",
			filename: "clip",
			content:  mp4,
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "file too small rejected",
			filename: "clip.mp4",
			content:  ftypHeader([]byte{0x00, 0x00, 0x00, 0x20}, "isom"),
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "file too large rejected",
			filename: "clip.mp4",
			content:  mp4,
			size:     maxSize + 1,
			wantErr:  ErrFileTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := tt.size
			if size == 0 {
				size = int64(len(tt.content))
			}
			limit := tt.maxSize
			if limit == 0 {
				limit = maxSize
			}

			file := newFile(tt.content)
			header := &multipart.FileHeader{Filename: tt.filename, Size: size}

			err := ValidateVideoFile(file, header, limit)
			switch {
			case tt.wantErr == nil && err != nil:
				t.Fatalf("ValidateVideoFile() unexpected error: %v", err)
			case tt.wantErr != nil && !errors.Is(err, tt.wantErr):
				t.Fatalf("ValidateVideoFile() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				// The file pointer must be rewound for the caller.
				pos, seekErr := file.Seek(0, 1)
				if seekErr != nil {
					t.Fatalf("Seek: %v", seekErr)
				}
				if pos != 0 {
					t.Errorf("file pointer = %d after validation, want 0", pos)
				}
			}
		})
	}
}

// Text that is not valid UTF-8 must be rejected here, as bad input.
//
// Regression test. Nothing checked encoding, so the bytes travelled all the way
// to Postgres, which refused them with SQLSTATE 22021 and turned a malformed
// title into a 500 INTERNAL_ERROR. A form posted in Windows-1252 is enough to
// cause it: an em dash arrives as the single byte 0x97, which no UTF-8 decoder
// will accept.
func TestValidateTextRejectsInvalidUTF8(t *testing.T) {
	// 0x97 is an em dash in Windows-1252 and an illegal continuation byte in UTF-8.
	cp1252EmDash := string([]byte{'A', 'u', 'r', 'o', 'r', 'a', ' ', 0x97, ' ', 'i', 'i'})

	if utf8.ValidString(cp1252EmDash) {
		t.Fatal("test fixture is valid UTF-8; it cannot exercise the bug")
	}

	if err := ValidateTitle(cp1252EmDash); !errors.Is(err, ErrInvalidTitle) {
		t.Errorf("ValidateTitle(invalid utf-8) = %v, want ErrInvalidTitle so the handler answers 400 rather than 500", err)
	}
	if err := ValidateDescription(cp1252EmDash); !errors.Is(err, ErrInvalidDescription) {
		t.Errorf("ValidateDescription(invalid utf-8) = %v, want ErrInvalidDescription", err)
	}
}

// Length is a count of characters, not bytes.
//
// `len(string)` counts bytes, so a title in a language that needs three bytes per
// character was refused at roughly 85 characters while the database column
// happily holds 255 of them.
func TestValidateTitleCountsRunesNotBytes(t *testing.T) {
	// 200 characters, 600 bytes. Well inside the limit, and the old byte-counting
	// check rejected it.
	title := strings.Repeat("あ", 200)

	if len(title) <= 255 {
		t.Fatal("fixture is under 255 bytes; it cannot exercise the bug")
	}
	if err := ValidateTitle(title); err != nil {
		t.Errorf("ValidateTitle(200 multi-byte chars) = %v, want nil — the limit is 255 characters, not bytes", err)
	}

	if err := ValidateTitle(strings.Repeat("あ", 256)); !errors.Is(err, ErrInvalidTitle) {
		t.Error("ValidateTitle(256 chars) = nil, want ErrInvalidTitle")
	}
}

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{name: "simple title", title: "My Video"},
		{name: "title with surrounding space", title: "   spaced   "},
		{name: "title at 255 chars", title: strings.Repeat("a", 255)},
		{name: "single char", title: "x"},
		{name: "empty", title: "", wantErr: true},
		{name: "whitespace only", title: "   \t\n  ", wantErr: true},
		{name: "256 chars too long", title: strings.Repeat("a", 256), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTitle(%q) error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidTitle) {
				t.Errorf("ValidateTitle(%q) error = %v, want ErrInvalidTitle", tt.title, err)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{name: "empty is allowed", description: ""},
		{name: "short", description: "A description."},
		{name: "exactly 5000", description: strings.Repeat("d", 5000)},
		{name: "5001 too long", description: strings.Repeat("d", 5001), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDescription(len=%d) error = %v, wantErr %v", len(tt.description), err, tt.wantErr)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	valid := "9a8b7c6d-5e4f-4a3b-8c1d-0e9f8a7b6c5d"

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid uuid", id: valid},
		{name: "valid uppercase uuid", id: strings.ToUpper(valid)},
		{name: "nil uuid string", id: "00000000-0000-0000-0000-000000000000"},
		{name: "empty", id: "", wantErr: true},
		{name: "not a uuid", id: "definitely-not-a-uuid", wantErr: true},
		{name: "truncated", id: "9a8b7c6d-5e4f-4a3b-8c1d", wantErr: true},
		{name: "bad character", id: "9a8b7c6d-5e4f-4a3b-8c1d-0e9f8a7b6cZZ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateUUID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUUID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidUUID) {
					t.Errorf("ValidateUUID(%q) error = %v, want ErrInvalidUUID", tt.id, err)
				}
				if got != uuid.Nil {
					t.Errorf("ValidateUUID(%q) = %v, want uuid.Nil on error", tt.id, got)
				}
				return
			}
			if !strings.EqualFold(got.String(), tt.id) {
				t.Errorf("ValidateUUID(%q) = %q, want round trip", tt.id, got.String())
			}
		})
	}
}

func TestValidatePageParams(t *testing.T) {
	tests := []struct {
		name    string
		page    int
		limit   int
		wantErr bool
	}{
		{name: "first page min limit", page: 1, limit: 1},
		{name: "typical", page: 3, limit: 20},
		{name: "max limit", page: 1, limit: 100},
		{name: "page zero", page: 0, limit: 10, wantErr: true},
		{name: "negative page", page: -1, limit: 10, wantErr: true},
		{name: "limit zero", page: 1, limit: 0, wantErr: true},
		{name: "negative limit", page: 1, limit: -5, wantErr: true},
		{name: "limit above 100", page: 1, limit: 101, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePageParams(tt.page, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePageParams(%d, %d) error = %v, wantErr %v", tt.page, tt.limit, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidPagination) {
				t.Errorf("ValidatePageParams(%d, %d) error = %v, want ErrInvalidPagination", tt.page, tt.limit, err)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no change", in: "clean", want: "clean"},
		{name: "leading and trailing spaces", in: "  hello  ", want: "hello"},
		{name: "tabs and newlines", in: "\t\nhello world\n\t", want: "hello world"},
		{name: "inner spacing preserved", in: "  a  b  ", want: "a  b"},
		{name: "whitespace only", in: " \t\n ", want: ""},
		{name: "empty", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeString(tt.in); got != tt.want {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
