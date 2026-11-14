# Writing documentation

Use plain English so readers can follow the guides without being fluent in English.

[Japanese](writing.ja.md)

## Writing guidelines

- Start with what the reader can do or needs to know.
- Use familiar words, such as “use”, “check”, and “run”.
- Give each sentence one main idea. Aim for 15–25 words; the lint limit is 35 words.
- Keep paragraphs short. Use a list for related features, requirements, or output fields.
- Use numbered steps when order matters.
- Use a table to compare options or explain fields.
- Put commands in fenced code blocks. State where to run them and list any prerequisites.
- Preserve exact command names, JSON fields, version numbers, and technical conditions.
- Explain an abbreviation when readers first encounter it.
- Prefer durable documentation about behavior, architecture, or development practices. Track temporary progress, coverage gaps, and planned work in GitHub Issues.

For example, replace a dense description with a short introduction and a list:

> Both commands output JSON with these fields:
>
> - `items`: update candidates
> - `bulk_actions`: actions for several updates
> - `unknown_controls`: unsupported controls

The [textlint README](https://github.com/textlint/textlint) and [Secretlint README](https://github.com/secretlint/secretlint) show features, setup steps, and command examples in separate sections.

## Examples and visual explanations

For complex topics, consider explaining the purpose, how it works, an example, and constraints.
Use this order to plan the explanation, not as a required template.
Use meaningful headings and avoid repeating the same explanation in several sections.
Avoid idioms, cultural references, and abstract descriptions when concrete terms explain the same idea.
Simplify language without losing technical accuracy.

Choose a visual format when it helps readers understand the content:

| Content | Preferred format |
| --- | --- |
| Execution flow | Mermaid flowchart |
| Component relationships | Mermaid or ASCII diagram |
| CLI syntax | Annotated command example |
| Input and output | Code blocks with expected results |
| Options and constraints | Markdown table |
| Valid and invalid cases | Comparison table |

Diagrams are optional. Keep Mermaid diagrams simple and compatible with GitHub Markdown.
Explain the key relationships in surrounding text so readers can understand them without viewing the diagram.
Check rendering when a renderer is available; otherwise, report that rendering remains unverified.

Use real renodash commands and behavior. Explain placeholders, input sources, expected results, and relevant error conditions.
Check examples against the implementation and specifications. Mark any unexecuted command examples as unverified.

### Annotated command example

With renodash installed, run this command from the repository root:

```sh
renodash parse --format json < testdata/renodash.md
```

Use a separate text block for annotations so readers can copy the command without copying the labels:

```text
renodash parse --format json < testdata/renodash.md
         |     |             | |
         |     |             | +-- Source Markdown file
         |     |             +-- Shell input redirection
         |     +-- Select JSON output (optional; the default)
         +-- Parse Markdown from standard input
```

The `<` operator belongs to the shell. It sends the file contents to renodash's standard input.
`--format json` is optional because JSON is the default and only supported output format.
The command writes JSON without fetching GitHub data or changing the file.
See [the parse example](usage.md#parse-saved-markdown) for expected output and the source-based command.

## Run the checks

The tools, rules, and lockfile belong to this repository. A global textlint installation is not required.

Install Node.js 22 or later and pnpm 12.16.1, as specified in [package.json](../package.json).
Run these commands from the repository root:

```sh
pnpm install --frozen-lockfile
pnpm run lint:docs
```

You can also use `make docs-lint`. Both `make lint` and `make check` include this check.

To apply fixes that the rules support:

```sh
pnpm run lint:docs:fix
```

Review each change. Some findings need a manual rewrite.

## Scope and rules

The check runs separate English and Japanese rules.

- English: the root README, English guides under `docs/`, the fixture README, example skills, and Markdown under `.agents/skills/`.
- Japanese: `readme_ja.md` and `docs/**/*.ja.md`.
- Saved Dashboard bodies and JSON snapshots are outside the scope.

Use `pnpm run lint:docs:en` or `pnpm run lint:docs:ja` to check one language.

English rules live in `.config/textlint/.textlintrc.json`:

- `en-max-word-count`: report sentences with more than 35 words.
- `write-good`: report repeated words, wordy phrases, and clichés.

Passive voice is allowed because technical references sometimes need it.
Checks for adverbs and vague words are disabled to keep feedback focused on useful edits.
The linter checks writing style. It does not verify technical facts or provide a complete grammar or spelling check.

Japanese rules live in `.config/textlint/.textlintrc.ja.json` and use `preset-ja-technical-writing`.
They check sentence length (up to 120 characters), repeated particles, usage errors, wordy phrases, and matching brackets.
Style consistency, long technical terms, and uncertainty expressions need manual review.

Place new English guides under `docs/` and name new Japanese guides `*.ja.md`.
Use `.config/textlint/.textlintignore` for saved Dashboard bodies and `.config/textlint/.textlintignore.en` to exclude Japanese documents from English checks.

Store behavior specifications under `docs/specs/`, mirroring implementation directories. See the [documentation index](README.md#specifications) for examples.
For when to add or update specifications, follow the [behavior change workflow](tdd.md#when-to-update-specifications).
