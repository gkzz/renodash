# Input validation

`get` checks the format of `OWNER/REPO#ISSUE_NUMBER` before calling the GitHub API.

- Local checks reject invalid paths and numbers.
- GitHub checks whether the account and repository exist and whether you have access.
- Passing local checks does not guarantee a successful fetch.
- `OWNER` is the account identifier in a URL, not a profile or organization display name.

## GitHub documentation reviewed

The original review covered GitHub.com on November 13, 2026.

| Topic | Findings and limits |
| --- | --- |
| Repository names | ASCII letters, digits, `-`, `_`, and `.`; up to 100 characters. See [Creating a new repository][repository]. |
| Personal accounts | The reviewed authentication guide describes letters, digits, and hyphens for Enterprise Managed Users. It does not establish all rules for regular personal accounts. |
| Managed users | Names can include a suffix, as in `mona-cat_octo`. The guide lists a 39-character limit including the suffix. See [Username considerations][usernames]. |
| Organizations | The reviewed sources do not provide a complete naming specification. [GitHub Pages troubleshooting][pages] discusses leading, trailing, and repeated hyphens. |

## Local rules

The input does not identify the account type. renodash does not add an API call to determine it.
Local owner checks allow a broad format to support existing accounts. They do not reproduce all GitHub account creation rules.

| Condition | `OWNER` | `REPO` |
| --- | --- | --- |
| Empty value | Reject | Reject |
| ASCII letters and digits | Allow | Allow |
| Hyphens and underscores | Allow at any position, including repeats | Allow at any position, including repeats |
| Dots | Reject | Allow, except names equal to `.` or `..` |
| Length | At least 1 byte; no local maximum | 1–100 bytes |
| Non-ASCII characters | Reject | Reject |
| Spaces, newlines, and control characters | Reject | Reject |
| Other symbols, such as `/`, `%`, `?`, and `#` | Reject | Reject |

- Allowed names use ASCII, so byte counts equal character counts.
- Rejecting dots in `OWNER` is a renodash design choice based on the reviewed username rules, not a complete organization naming specification.
- Owner length and hyphen positions remain unrestricted until evidence covers each account type.
- Underscores support managed user names. Their position and suffix format are not checked.
- Rejecting repository names `.` and `..` avoids special path components.
- `...` remains allowed because the review found no official reason to reject it.
- Suffixes such as `.git` and `.wiki` are neither rejected nor removed.
- Local checks do not reproduce reserved names, name availability checks, or name changes made by GitHub's creation form.

## Examples and limits

| Input | Local result | Reason |
| --- | --- | --- |
| `octocat/Hello-World#1` | Pass | Standard format |
| `octo-org/repo#2` | Pass | Organization format |
| `org-/repo#1` | Pass | Trailing owner hyphens are allowed |
| `org--name/repo#1` | Pass | Repeated owner hyphens are allowed |
| `mona-cat_octo/repo#1` | Pass | Managed user format |
| `owner/._repo--_.#1` | Pass | Repository symbols can appear at any position |
| `owner/...#1` | Pass | Different from `.` and `..` |
| `OWNER/Repo#0001` | Pass | Keeps name casing; converts the number to 1 |
| `owner.name/repo#1` | Fail | Dot in owner name |
| `owner/..#1` | Fail | Special path component |
| `owner/répo#1` | Fail | Non-ASCII character |
| `owner/repo%2fother#1` | Fail | URL encoding is not decoded |
| `owner/repo#0` | Fail | Issue number must be positive |

- Repository names of 1 and 100 characters pass; 101 characters fail.
- A 40-character owner passes locally. This does not establish whether GitHub accepts it.
- Issue numbers can reach the platform's `int` limit. Larger values fail.

## How `parseIssueRef` works

1. Use `strings.Cut` to split at the first `#`.
2. Split the first part at `/` and require exactly two parts: owner and repository.
3. Check the names with `validOwner` and `validRepo`.
4. Require a nonempty issue number containing only ASCII digits `0`–`9`.
5. Convert it to `int` with `strconv.Atoi`. Reject conversion errors and values below 1.

| Case | Behavior |
| --- | --- |
| Leading or trailing spaces | Reject; no trimming |
| Full URLs or extra `/` or `#` characters | Reject |
| Signs or full-width digits in issue numbers | Reject |
| Leading zeros | Allow |
| Success | Return `OWNER/REPO` with its original casing, the issue number, and `nil` |
| Failure | Return an empty string, 0, and an error |

The CLI syntax and existing error messages remain unchanged.
URL matching and metadata checks after fetching remain separate.

## Implementation and tests

- Implementation: `parseIssueRef`, `validOwner`, and `validRepo` in [validation.go](../../../../internal/command/validation.go).
- Tests: `TestParseIssueRef` in [validation_test.go](../../../../internal/command/validation_test.go) covers name formats, character and length limits, and issue number boundaries.

## Reading the Go code

`validOwner` and `validRepo` handle different name rules. Both share this character check:

```go
func asciiAlphanumeric(c rune) bool {
    return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
```

- `bool` represents `true` or `false`.
- Name checks return a boolean. `parseIssueRef` builds the user-facing error.
- `!validOwner(parts[0])` means the owner is invalid.

```go
for _, c := range owner {
    if !(asciiAlphanumeric(c) || c == '-' || c == '_') {
        return false
    }
}
```

| Go syntax | Meaning |
| --- | --- |
| `range` over a string | Returns byte positions and Unicode code points as `rune` values |
| `_` | Discards the unused byte position |
| `rune` | Alias for `int32`; can represent non-ASCII characters, though this check rejects them |
| `!` | Logical NOT |
| `&&` | Logical AND; skips the right side when the left side is false |
| `\|\|` | Logical OR; skips the right side when the left side is true |

Invalid UTF-8 also fails the character check.
Operator precedence is NOT, comparison, AND, then OR. Parentheses can make groups explicit.
In `len(parts) != 2 || !validOwner(parts[0])`, skipping the right side prevents access to a missing element.

```go
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
```

| Test syntax | Meaning |
| --- | --- |
| `tests` | Slice of structs containing case names, inputs, and expected results |
| `t.Run` | Runs a named subtest for each case |
| `t.Fatalf` | Records a failure and stops the current subtest |
| `t.Errorf` | Records a failure and continues the subtest |
| `if gotErr := ...; ...` | Declares `gotErr` within the `if` statement |
| `%q`, `%t`, `%v` | Format a quoted string, boolean, and default value representation |

## Follow-up work and test scope

- Confirm official length and symbol rules for organizations and regular personal accounts.
- Assess effects on existing account references before adding restrictions.
- Tests cover offline format checks and fetching through a fake `gh` command.
- Tests do not cover account creation or connections to real repositories.

[repository]: https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-new-repository
[usernames]: https://docs.github.com/en/enterprise-cloud%40latest/admin/managing-iam/iam-configuration-reference/username-considerations-for-external-authentication
[pages]: https://docs.github.com/en/enterprise-cloud%40latest/pages/configuring-a-custom-domain-for-github-pages-site/troubleshooting-custom-domains-and-github-pages
