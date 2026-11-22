# Architecture Decision Records

Architecture Decision Records (ADRs) explain what renodash chooses and why.
Use them for technical decisions whose context, alternatives, and consequences will help future developers.
Keep developer procedures in the [development guide](../development.md) and behavior contracts in the [specifications](../README.md#specifications).

## Conventions

- Store each decision in this directory as `NNNN-short-description.md`, using the next unused four-digit number.
- Include a title, status, date, context, decision drivers, considered options, decision, consequences, and evidence or references.
- Start with `Proposed`. Change to `Accepted` only when maintainer approval of the decision is recorded; link to that approval.
- Use `Rejected` for a declined proposal. Use `Superseded` when a later accepted ADR replaces an accepted decision, and link the records in both directions.
- Update this index when adding a record or changing its status. Preserve earlier decisions so readers can understand their context.

An implementation branch can follow a proposed ADR for review without implying approval.
The status records approval of the decision, rather than implementation progress.

## Records

| ADR | Title | Status |
| --- | --- | --- |
| 0001 | [Go test comparison strategy](0001-go-test-comparison-strategy.md) | Accepted |
