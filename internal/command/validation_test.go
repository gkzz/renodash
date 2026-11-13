package command

import (
	"strconv"
	"strings"
	"testing"
)

// TestParseIssueRef checks the syntax and return values for OWNER/REPO#ISSUE_NUMBER.
// It validates owner and repository characters, the 100-character repository limit, positive integer bounds,
// and leading zeros. Invalid input returns an error, an empty repository name, and issue number zero.
func TestParseIssueRef(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name       string
		input      string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{
			name:       "personal account",
			input:      "octocat/Hello-World#1",
			wantRepo:   "octocat/Hello-World",
			wantNumber: 1,
		},
		{
			name:       "organization",
			input:      "octo-org/repo#2",
			wantRepo:   "octo-org/repo",
			wantNumber: 2,
		},
		{
			name:       "owner trailing hyphen",
			input:      "org-/repo#1",
			wantRepo:   "org-/repo",
			wantNumber: 1,
		},
		{
			name:       "owner consecutive hyphens",
			input:      "org--name/repo#1",
			wantRepo:   "org--name/repo",
			wantNumber: 1,
		},
		{
			name:       "owner leading hyphen compatibility",
			input:      "-org/repo#1",
			wantRepo:   "-org/repo",
			wantNumber: 1,
		},
		{
			name:       "managed user",
			input:      "mona-cat_octo/repo#1",
			wantRepo:   "mona-cat_octo/repo",
			wantNumber: 1,
		},
		{
			name:       "owner length not restricted locally",
			input:      strings.Repeat("a", 40) + "/repo#1",
			wantRepo:   strings.Repeat("a", 40) + "/repo",
			wantNumber: 1,
		},
		{
			name:    "repository punctuation",
			input:   "owner/._repo--_ .#1",
			wantErr: true,
		},
		{
			name:       "repository leading punctuation",
			input:      "owner/._repo--_.#1",
			wantRepo:   "owner/._repo--_.",
			wantNumber: 1,
		},
		{
			name:       "repository triple dot",
			input:      "owner/...#1",
			wantRepo:   "owner/...",
			wantNumber: 1,
		},
		{
			name:       "repository git suffix retained",
			input:      "owner/repo.git#1",
			wantRepo:   "owner/repo.git",
			wantNumber: 1,
		},
		{
			name:       "repository wiki suffix retained",
			input:      "owner/repo.wiki#1",
			wantRepo:   "owner/repo.wiki",
			wantNumber: 1,
		},
		{
			name:       "ASCII digits",
			input:      "123/456#1",
			wantRepo:   "123/456",
			wantNumber: 1,
		},
		{
			name:    "NUL",
			input:   "owner/repo\x00#1",
			wantErr: true,
		},
		{
			name:       "maximum int",
			input:      "owner/repo#" + strconv.Itoa(maxInt),
			wantRepo:   "owner/repo",
			wantNumber: maxInt,
		},
		{
			name:       "repository hyphen",
			input:      "owner/-#1",
			wantRepo:   "owner/-",
			wantNumber: 1,
		},
		{
			name:       "repository underscore",
			input:      "owner/_#1",
			wantRepo:   "owner/_",
			wantNumber: 1,
		},
		{
			name:       "repository maximum",
			input:      "owner/" + strings.Repeat("a", 100) + "#1",
			wantRepo:   "owner/" + strings.Repeat("a", 100),
			wantNumber: 1,
		},
		{
			name:    "repository too long",
			input:   "owner/" + strings.Repeat("a", 101) + "#1",
			wantErr: true,
		},
		{
			name:    "owner dot",
			input:   "owner.name/repo#1",
			wantErr: true,
		},
		{
			name:    "owner triple dot",
			input:   ".../repo#1",
			wantErr: true,
		},
		{
			name:    "empty owner",
			input:   "/repo#1",
			wantErr: true,
		},
		{
			name:    "empty repository",
			input:   "owner/#1",
			wantErr: true,
		},
		{
			name:    "repository dot",
			input:   "owner/.#1",
			wantErr: true,
		},
		{
			name:    "repository double dot",
			input:   "owner/..#1",
			wantErr: true,
		},
		{
			name:    "unicode owner",
			input:   "所有者/repo#1",
			wantErr: true,
		},
		{
			name:    "unicode repository",
			input:   "owner/リポジトリ#1",
			wantErr: true,
		},
		{
			name:    "non ASCII letter",
			input:   "owner/café#1",
			wantErr: true,
		},
		{
			name:    "space",
			input:   "owner/re po#1",
			wantErr: true,
		},
		{
			name:    "control",
			input:   "owner/repo\n#1",
			wantErr: true,
		},
		{
			name:    "escaped path",
			input:   "owner/repo%2fother#1",
			wantErr: true,
		},
		{
			name:    "query",
			input:   "owner/repo?x#1",
			wantErr: true,
		},
		{
			name:    "missing separator",
			input:   "owner/repo",
			wantErr: true,
		},
		{
			name:    "extra slash",
			input:   "owner/repo/other#1",
			wantErr: true,
		},
		{
			name:    "empty number",
			input:   "owner/repo#",
			wantErr: true,
		},
		{
			name:    "zero",
			input:   "owner/repo#0",
			wantErr: true,
		},
		{
			name:    "negative",
			input:   "owner/repo#-1",
			wantErr: true,
		},
		{
			name:    "plus sign",
			input:   "owner/repo#+1",
			wantErr: true,
		},
		{
			name:    "unicode digit",
			input:   "owner/repo#１",
			wantErr: true,
		},
		{
			name:    "extra hash",
			input:   "owner/repo#1#2",
			wantErr: true,
		},
		{
			name:    "overflow",
			input:   "owner/repo#" + strings.Repeat("9", 100),
			wantErr: true,
		},
		{
			name:       "leading zeros",
			input:      "OWNER/Repo#0001",
			wantRepo:   "OWNER/Repo",
			wantNumber: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, number, err := parseIssueRef(tt.input)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("parseIssueRef(%q) error=%v (has error=%t), want has error=%t", tt.input, err, gotErr, tt.wantErr)
			}
			if repo != tt.wantRepo {
				t.Errorf("parseIssueRef(%q) repository=%q, want %q", tt.input, repo, tt.wantRepo)
			}
			if number != tt.wantNumber {
				t.Errorf("parseIssueRef(%q) issue number=%d, want %d", tt.input, number, tt.wantNumber)
			}
		})
	}
}
