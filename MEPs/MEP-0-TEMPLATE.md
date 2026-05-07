# MEP-0: MURMUR Extension Proposal Template

## Title

[One-line summary of the proposed extension]

## Status

Proposed

## Motivation

[Why is this extension needed? What problem does it solve? What user need does it address?]

## Proposed Interface

### API Changes (if applicable)

[Describe new interfaces, types, functions, or methods. Use code blocks for clarity.]

```go
type ExampleInterface interface {
    Method1(...) error
    Method2(...) (result, error)
}
```

### Protocol Changes (if applicable)

[Describe new message types, topics, or stream protocols.]

```protobuf
message ExampleMessage {
    uint32 field1 = 1;
    bytes field2 = 2;
}
```

### Behavior Changes (if applicable)

[Describe how existing behavior changes, if at all.]

## Stability

Select one:

- **Stable**: Fully designed, backward compatible, ready for long-term use
- **Experimental**: May change based on feedback; recommend feature-gating in early stages
- **Private**: Not yet exposed to third parties; internal use only

[Your choice here]: [Explain the reasoning]

## Backward Compatibility

[How do old clients interact with this extension?]

- Old clients **ignore** messages of the new type without error?
- Old clients **fail gracefully** when encountering the extension?
- Old clients **require an update** to use this extension?

[Describe the compatibility guarantees and any deprecation schedule.]

## Security Considerations

[What new threat surfaces does this introduce?]

- New message types: Does the extension expose new information?
- New transport: Does the extension leak metadata?
- New protocols: Could the extension be abused?
- Key management: Are any new secrets introduced?

[Reference SECURITY_PRIVACY.md and THREAT_MODEL.md as applicable.]

## Design & Rationale

[Why this particular design? What alternatives were considered and rejected?]

## Implementation Notes

[Hints for implementers. Which modules must be modified? Are there gotchas?]

## Example Usage

[Show how a client or third party would use this extension.]

```go
// Example code
module := mypackage.NewModule()
if err := mechanics.RegisterGameModule(module); err != nil {
    // Handle error
}
```

## Testing Strategy

[How will this extension be tested? Unit tests? Integration tests?]

## Risks & Mitigations

[What could go wrong? How will you mitigate risks?]

| Risk | Mitigation |
|---|---|
| ... | ... |

## Related Work

[Links to relevant specs, MEPs, or discussions]

- EXTENSION_CONTRACT.md
- PROTOCOL.md
- [Issue #123](relative issue link)

## Questions for Reviewers

[Open questions or areas where you want community feedback]

- Question 1?
- Question 2?

---

**Author**: [Your name or GitHub handle]  
**Created**: [Date]  
**Last Updated**: [Date]
