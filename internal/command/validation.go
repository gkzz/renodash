package command

import (
	"errors"
	"strconv"
	"strings"
)

func parseIssueRef(value string) (string, int, error) {
	repo, numberText, ok := strings.Cut(value, "#")
	parts := strings.Split(repo, "/")
	if !ok || len(parts) != 2 || !validOwner(parts[0]) || !validRepo(parts[1]) || numberText == "" {
		return "", 0, errors.New("invalid issue reference: expected OWNER/REPO#ISSUE_NUMBER")
	}
	for _, c := range numberText {
		if c < '0' || c > '9' {
			return "", 0, errors.New("invalid issue reference: issue number must be a positive integer")
		}
	}
	number, err := strconv.Atoi(numberText)
	if err != nil || number <= 0 {
		return "", 0, errors.New("invalid issue reference: issue number must be a positive integer")
	}
	return repo, number, nil
}

// validOwner checks a permissive format shared by user and organization owners.
// Do not apply personal-account hyphen rules to organizations or reject managed users.
func validOwner(owner string) bool {
	if owner == "" {
		return false
	}
	for _, c := range owner {
		if !(asciiAlphanumeric(c) || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func asciiAlphanumeric(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func validRepo(part string) bool {
	if part == "" || len(part) > 100 || part == "." || part == ".." {
		return false
	}
	for _, c := range part {
		if !(asciiAlphanumeric(c) || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
