# Architecture Decision Records (ADR)

This directory contains Architecture Decision Records (ADRs) for the k8s-chaos project.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences. ADRs help teams:

- Understand why decisions were made
- Onboard new team members faster
- Avoid revisiting settled decisions
- Learn from past choices
- Document trade-offs and alternatives

## ADR Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0000](0000-adr-template.md) | ADR Template | Template | - |
| [0001](0001-crd-validation-strategy.md) | CRD Validation Strategy | Accepted | 2025-10-10 |
| [0002](0002-safety-features-implementation.md) | Safety Features Implementation | Accepted | 2025-11-04 |
| [0003](0003-pod-memory-stress-implementation.md) | Pod Memory Stress Implementation | Accepted | 2025-11-07 |
| [0004](0004-pod-failure-implementation.md) | Pod Failure Implementation | Accepted | 2025-11-19 |
| [0005](0005-pod-cpu-stress-implementation.md) | Pod CPU Stress Implementation | Accepted | 2025-10-28 |

## ADR Lifecycle

```
Proposed → Accepted → [Deprecated | Superseded]
```

- **Proposed**: Under discussion, not yet implemented
- **Accepted**: Decision made and being/been implemented
- **Deprecated**: No longer relevant but kept for historical context
- **Superseded**: Replaced by a newer ADR

## Creating a New ADR

1. **Copy the template**:
   ```bash
   cp docs/adr/0000-adr-template.md docs/adr/XXXX-your-decision.md
   ```

2. **Use sequential numbering**: Find the highest numbered ADR and increment by 1

3. **Choose a descriptive name**: Use lowercase with hyphens
   - Good: `0002-metrics-storage-backend.md`
   - Bad: `adr-2.md`, `decision.md`

4. **Fill in all sections**: Don't skip sections, they all add value

5. **Start with "Proposed"**: Status should be "Proposed" until team review

6. **Create a PR**: ADRs should be reviewed before being accepted

7. **Update this README**: Add your ADR to the index table above

## ADR Best Practices

### Do:
- ✅ Write ADRs for significant architectural decisions
- ✅ Keep ADRs concise but complete
- ✅ Include code examples when relevant
- ✅ Document alternatives considered
- ✅ Be honest about trade-offs and consequences
- ✅ Update implementation status as work progresses
- ✅ Reference related issues, PRs, and documentation

### Don't:
- ❌ Modify accepted ADRs (create new ones that supersede)
- ❌ Skip the "Alternatives Considered" section
- ❌ Forget to document negative consequences
- ❌ Write overly detailed implementation guides (link to docs instead)
- ❌ Use ADRs for trivial decisions
- ❌ Let ADRs become stale - update implementation status

## When to Write an ADR

Write an ADR when:
- Choosing between multiple technical approaches
- Making decisions that impact multiple components
- Establishing patterns that other developers should follow
- Making trade-offs that future developers should understand
- Choosing dependencies or external services
- Defining API contracts or data models

Don't write an ADR for:
- Routine bug fixes
- Minor refactoring
- Updating dependencies to latest versions
- Documentation improvements
- Test additions

## ADR Structure

Our ADRs follow this structure:

1. **Title & Metadata**: Status, date, authors
2. **Context**: Problem statement and requirements
3. **Decision**: The chosen solution with key details
4. **Alternatives Considered**: What else was evaluated and why rejected
5. **Consequences**: Positive, negative, and neutral impacts
6. **Implementation Status**: What's done and what's planned
7. **References**: Links to related resources
8. **Notes**: Additional context or open questions

## Examples

### Good ADR Titles
- "Use OpenAPI Schema Validation for CRD Fields"
- "Implement Leader Election for Controller HA"
- "Choose Prometheus for Metrics Collection"
- "Adopt Envtest for Controller Unit Testing"

### Poor ADR Titles
- "Validation" (too vague)
- "Fix the controller" (not a decision)
- "Update to latest Go version" (routine maintenance)

## Superseding ADRs

When a decision needs to change:

1. Create a new ADR with the updated decision
2. Update the old ADR's status to "Superseded by ADR-XXXX"
3. Explain why the original decision is being changed
4. Keep the old ADR for historical context

Example:
```markdown
# ADR 0001: Use REST API for Chaos Injection

**Status:** Superseded by ADR-0015

**Superseded on:** 2025-11-15

**Reason:** Performance issues at scale led us to adopt gRPC instead.
```

## Related Resources

- [ADR GitHub Organization](https://adr.github.io/)
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [Architecture Decision Records in Action](https://www.youtube.com/watch?v=41NVge3_cYo)
- [Kubernetes Enhancement Proposals (KEPs)](https://github.com/kubernetes/enhancements) - Similar concept

## Contributing

ADRs should be reviewed as part of the normal PR process. When reviewing:

- Check that all sections are complete
- Verify alternatives were properly considered
- Ensure consequences are realistic
- Confirm the decision aligns with project goals
- Suggest improvements to clarity and completeness

ADRs are living documents - update implementation status as work progresses!
