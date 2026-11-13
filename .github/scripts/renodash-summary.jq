# Display GitHub observations without evaluating merge readiness.
def safe:
  tostring
  | gsub("&"; "&amp;")
  | gsub("<"; "&lt;")
  | gsub(">"; "&gt;")
  | gsub("\\\\"; "&#92;")
  | gsub("\\*"; "&#42;")
  | gsub("_"; "&#95;")
  | gsub("\\["; "&#91;")
  | gsub("\\]"; "&#93;")
  | gsub("[\r\n]+"; " ")
  | gsub("\\|"; "&#124;")
  | gsub("`"; "&#96;");

"## Renovate Dependency Dashboard (saved GitHub snapshots)",
"",
"PR states and checks are real GitHub responses captured and saved in advance for offline testing. They reflect capture time, not current GitHub state.",
"",
"The Dashboard body is constructed for testing with real PR references; it is not a captured live Dashboard.",
"See testdata/pr-metadata/README.md for source URLs, capture time, and provenance.",
"",
"- **Issue:** [#\(.source.issue_number)](\(.source.issue_url))",
"- **Test execution:** \(.source.fetched_at)",
"- **Candidates:** \(.items | length)",
"- **Bulk actions:** \(.bulk_actions | length)",
"- **Unknown controls:** \(.unknown_controls | length)",
"- **Linked PRs:** \(.pull_requests | length)",
"",
"### Update candidates",
"",
"| Section | Action | Title |",
"| --- | --- | --- |",
(
  if (.items | length) == 0 then
    "| - | - | No candidates |"
  else
    .items[:20][]
    | "| \(.section | safe) | \(.action | safe) | \(.title | safe) |"
  end
),
(
  if (.items | length) > 20 then
    "",
    "_Showing the first 20 candidates. Download the artifact for the full results._"
  else
    empty
  end
),
"",
"### Linked pull requests",
"",
if (.pull_requests | length) == 0 then
  "No linked pull requests were found in the Dependency Dashboard. PR metadata verification was skipped."
else
  "| PR | State | Draft | Title | Checks |",
  "| --- | --- | --- | --- | --- |",
  (.pull_requests[] |
    "| [#\(.number)](\(.url)) | \(.state | safe) | \(.is_draft) | \(.title | safe) | \(
      if (.checks | length) == 0 then
        "Unknown (no checks returned)"
      else
        [.checks[] |
          ((.name // .context // "Unnamed check") | safe) + ": " +
          (([.conclusion, .status, .state] | map(select(. != null and . != "")) | first // "Unknown") | safe)
        ] | join("<br>")
      end
    ) |"),
  "",
  "Check results are observations, not merge approval."
end
