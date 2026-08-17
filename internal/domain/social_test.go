package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func intPtr(v int) *int { return &v }

func TestCommentValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Comment)
		wantErr error
	}{
		{
			name:   "a plain comment",
			mutate: func(*Comment) {},
		},
		{
			name:   "anchored to a moment",
			mutate: func(c *Comment) { c.VideoTimestamp = intPtr(94) },
		},
		{
			// Zero is the first frame, and is distinct from "no anchor". It
			// must survive validation rather than being treated as unset.
			name:   "anchored to the first frame",
			mutate: func(c *Comment) { c.VideoTimestamp = intPtr(0) },
		},
		{
			name:    "anchored before the start",
			mutate:  func(c *Comment) { c.VideoTimestamp = intPtr(-1) },
			wantErr: ErrInvalidInput,
		},
		{
			name:    "no author",
			mutate:  func(c *Comment) { c.UserID = uuid.Nil },
			wantErr: ErrInvalidInput,
		},
		{
			name:    "no video",
			mutate:  func(c *Comment) { c.VideoID = uuid.Nil },
			wantErr: ErrInvalidInput,
		},
		{
			name:    "empty content",
			mutate:  func(c *Comment) { c.Content = "" },
			wantErr: ErrInvalidInput,
		},
		{
			name:    "content past the column limit",
			mutate:  func(c *Comment) { c.Content = strings.Repeat("a", 10001) },
			wantErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := &Comment{
				ID:      uuid.New(),
				VideoID: uuid.New(),
				UserID:  uuid.New(),
				Content: "looks good",
			}
			tt.mutate(comment)

			if err := comment.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommentSortIsValid(t *testing.T) {
	valid := []CommentSort{CommentSortNewest, CommentSortTimestamp}
	for _, sort := range valid {
		if !sort.IsValid() {
			t.Errorf("%q should be a valid sort", sort)
		}
	}

	// The value reaches the repository, which picks an ORDER BY from it. Only a
	// known value may get that far.
	for _, sort := range []CommentSort{"", "oldest", "created_at DESC; DROP TABLE comments"} {
		if CommentSort(sort).IsValid() {
			t.Errorf("%q should not be a valid sort", sort)
		}
	}
}
