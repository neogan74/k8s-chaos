# Contributing to k8s-chaos

Thank you for your interest in contributing to k8s-chaos! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [How to Contribute](#how-to-contribute)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Issue Guidelines](#issue-guidelines)
- [Architecture Decision Records](#architecture-decision-records)
- [Community](#community)

## Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow. Please be respectful, inclusive, and constructive in all interactions.

### Our Standards

- **Be respectful**: Value diverse perspectives and experiences
- **Be collaborative**: Work together towards common goals
- **Be constructive**: Provide helpful feedback and solutions
- **Be inclusive**: Welcome newcomers and help them contribute
- **Be professional**: Keep discussions focused and on-topic

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- **Go 1.24.5+**: [Install Go](https://golang.org/doc/install)
- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **kubectl**: [Install kubectl](https://kubernetes.io/docs/tasks/tools/)
- **Kind**: For local Kubernetes clusters
- **Git**: For version control
- **Make**: For build automation

### Quick Setup

```bash
# 1. Fork the repository on GitHub
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/k8s-chaos.git
cd k8s-chaos

# 3. Add upstream remote
git remote add upstream https://github.com/neogan74/k8s-chaos.git

# 4. Set up development environment
make dev-setup

# 5. Verify everything works
make test
```

## Development Environment

### Automated Setup

The fastest way to get started:

```bash
make dev-setup
```

This creates:
- Kind cluster with multi-node configuration
- CRDs installed
- Demo environment with test pods
- All development tools

### Manual Setup

If you prefer manual setup:

```bash
# Create Kind cluster
kind create cluster --name k8s-chaos-dev

# Install CRDs
make install

# Run controller locally
make run
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for detailed development guide.

## How to Contribute

### Ways to Contribute

We welcome various types of contributions:

- **Bug Reports**: Found a bug? Report it!
- **Feature Requests**: Have an idea? Suggest it!
- **Code Contributions**: Fix bugs or implement features
- **Documentation**: Improve or add documentation
- **Testing**: Write tests or improve test coverage
- **Examples**: Add example chaos experiments
- **Tutorials**: Create learning materials
- **Translations**: Translate documentation (future)

### Good First Issues

Look for issues tagged with `good first issue` - these are suitable for newcomers:

```bash
# View good first issues
gh issue list --label "good first issue"
```

### What to Work On

Before starting work:

1. **Check existing issues**: Avoid duplicate work
2. **Discuss large changes**: Open an issue for discussion first
3. **Ask questions**: Don't hesitate to ask for clarification
4. **Start small**: Begin with small contributions to learn the codebase

## Development Workflow

### 1. Create a Branch

```bash
# Update your fork
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/bug-description
```

### Branch Naming Convention

- `feature/feature-name` - New features
- `fix/bug-description` - Bug fixes
- `docs/what-changed` - Documentation changes
- `refactor/what-refactored` - Code refactoring
- `test/what-tested` - Test additions/improvements

### 2. Make Changes

```bash
# Make your changes
vim internal/controller/chaosexperiment_controller.go

# If you changed API types, regenerate
make manifests generate

# Format code
make fmt

# Run static analysis
make vet

# Run linter
make lint

# Run tests
make test
```

### 3. Commit Changes

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Commit format
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Build process, dependencies

**Examples:**

```bash
git commit -m "feat(controller): add pod-network-loss action"

git commit -m "fix(webhook): validate cron expression format

- Add regex validation for cron schedules
- Return clear error messages for invalid formats
- Add unit tests for schedule validation

Fixes #123"

git commit -m "docs(contributing): add PR process guidelines"
```

### 4. Push Changes

```bash
# Push to your fork
git push origin feature/your-feature-name
```

### 5. Create Pull Request

1. Go to GitHub and create a pull request
2. Fill in the PR template
3. Link related issues
4. Request reviews
5. Address review feedback

## Code Standards

### Go Code Style

Follow standard Go conventions:

- **gofmt**: All code must be formatted with `gofmt`
- **golangci-lint**: Must pass all linter checks
- **go vet**: Must pass static analysis
- **Effective Go**: Follow [Effective Go](https://golang.org/doc/effective_go) guidelines

```bash
# Format code
make fmt

# Run linter
make lint

# Run vet
make vet
```

### Code Organization

```
├── api/v1alpha1/          # API definitions
│   ├── *_types.go         # Type definitions
│   ├── *_webhook.go       # Webhook logic
│   └── *_test.go          # API tests
├── internal/controller/   # Controller logic
│   ├── *.go              # Implementation
│   └── *_test.go         # Controller tests
├── internal/metrics/      # Metrics definitions
├── cmd/                   # Main entrypoints
├── config/               # Kubernetes manifests
└── docs/                 # Documentation
```

### Naming Conventions

- **Files**: lowercase, underscores (e.g., `chaosexperiment_controller.go`)
- **Types**: PascalCase (e.g., `ChaosExperiment`)
- **Functions**: camelCase (private), PascalCase (public)
- **Constants**: PascalCase or UPPER_CASE for values
- **Variables**: camelCase

### Comments and Documentation

```go
// Package controller implements the ChaosExperiment controller.
package controller

// ChaosExperimentReconciler reconciles a ChaosExperiment object.
// It implements the controller-runtime Reconciler interface.
type ChaosExperimentReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

// Reconcile implements the reconciliation loop for ChaosExperiment resources.
// It handles experiment scheduling, execution, and status updates.
//
// The reconciliation logic:
// 1. Fetches the ChaosExperiment resource
// 2. Checks if execution is needed (schedule/duration)
// 3. Selects eligible target resources
// 4. Executes the chaos action
// 5. Updates status and creates history
// 6. Requeues for next execution
func (r *ChaosExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Implementation
}
```

### Error Handling

```go
// Good: Descriptive error messages
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
}

// Good: Use errors.Is() for error checking
if errors.Is(err, ErrNoPodsFound) {
    // Handle specific error
}

// Bad: Generic error messages
if err != nil {
    return ctrl.Result{}, err
}
```

### Logging

Use structured logging with controller-runtime:

```go
logger := log.FromContext(ctx)

// Good: Structured logging
logger.Info("executing chaos action",
    "action", exp.Spec.Action,
    "namespace", exp.Spec.Namespace,
    "count", count)

// Bad: Unstructured logging
logger.Info(fmt.Sprintf("Executing %s on %d pods", action, count))
```

## Testing

### Test Requirements

All contributions must include appropriate tests:

- **Unit Tests**: For all new functions
- **Integration Tests**: For controller logic
- **E2E Tests**: For complete workflows (if applicable)

### Running Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
make test
go tool cover -html=cover.out

# Run specific package tests
go test ./internal/controller -v

# Run E2E tests (requires Docker)
make test-e2e

# Run specific test
go test ./internal/controller -run TestReconcile -v
```

### Writing Tests

Use Ginkgo/Gomega for tests:

```go
package controller_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("ChaosExperiment Controller", func() {
    Context("When reconciling a pod-kill experiment", func() {
        It("Should delete the specified number of pods", func() {
            // Setup
            exp := &v1alpha1.ChaosExperiment{
                Spec: v1alpha1.ChaosExperimentSpec{
                    Action: "pod-kill",
                    Count: ptr.To(int32(2)),
                },
            }

            // Execute
            result, err := reconciler.Reconcile(ctx, req)

            // Assert
            Expect(err).NotTo(HaveOccurred())
            Expect(result.Requeue).To(BeTrue())
        })
    })
})
```

### Test Coverage

- Aim for **80%+ code coverage**
- Critical paths must have **100% coverage**
- Add tests before fixing bugs (TDD)

## Documentation

### Types of Documentation

1. **Code Comments**: Document complex logic
2. **API Documentation**: Godoc for public APIs
3. **User Documentation**: Markdown files in `docs/`
4. **Examples**: Sample CRDs in `config/samples/`
5. **ADRs**: Architecture decisions in `docs/adr/`

### Documentation Standards

- **Clear and concise**: Easy to understand
- **Examples included**: Show, don't just tell
- **Up to date**: Update docs with code changes
- **Properly formatted**: Use markdown properly

### Updating Documentation

When changing functionality:

```bash
# 1. Update relevant docs
vim docs/API.md
vim docs/GETTING-STARTED.md

# 2. Update examples if needed
vim config/samples/chaos_v1alpha1_chaosexperiment.yaml

# 3. Update CHANGELOG (if applicable)
vim CHANGELOG.md

# 4. Include in PR
git add docs/ config/samples/
git commit -m "docs: update API documentation for new feature"
```

### Writing ADRs

For significant architectural decisions:

```bash
# 1. Copy template
cp docs/adr/0000-adr-template.md docs/adr/XXXX-your-decision.md

# 2. Fill in all sections
# 3. Create PR for review
# 4. Update docs/adr/README.md
```

See [docs/adr/README.md](docs/adr/README.md) for guidelines.

## Pull Request Process

### Before Opening a PR

Checklist:

- [ ] Code is formatted (`make fmt`)
- [ ] Linter passes (`make lint`)
- [ ] Tests pass (`make test`)
- [ ] Documentation updated
- [ ] Examples added/updated (if applicable)
- [ ] CHANGELOG updated (for user-facing changes)
- [ ] Commits follow conventional commit format

### PR Template

Fill in the PR template completely:

```markdown
## Description
Brief description of changes.

## Related Issue
Fixes #123

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring

## Testing
Describe how you tested the changes.

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Examples added
- [ ] CHANGELOG updated
```

### PR Review Process

1. **Automated Checks**: GitHub Actions must pass
   - Unit tests
   - Linter checks
   - Build verification
   - E2E tests

2. **Code Review**: At least one approval required
   - Reviewers may request changes
   - Address feedback in new commits
   - Don't force-push during review

3. **Final Approval**: Maintainer approves and merges
   - Squash merge for feature branches
   - Preserve commit history for special cases

### Review Feedback

When receiving feedback:

```bash
# Make changes based on feedback
vim internal/controller/chaosexperiment_controller.go

# Commit with clear message
git commit -m "review: address feedback on error handling"

# Push updates
git push origin feature/your-feature-name

# Respond to comments on GitHub
```

## Issue Guidelines

### Reporting Bugs

Use the bug report template:

```markdown
**Describe the bug**
Clear description of the issue.

**To Reproduce**
Steps to reproduce:
1. Create experiment with...
2. Apply to cluster...
3. Observe error...

**Expected behavior**
What should happen.

**Actual behavior**
What actually happens.

**Environment**
- k8s-chaos version: v0.1.0
- Kubernetes version: v1.28.0
- Installation method: Helm

**Additional context**
Logs, screenshots, etc.
```

### Requesting Features

Use the feature request template:

```markdown
**Feature Description**
Clear description of the feature.

**Use Case**
Why is this needed? What problem does it solve?

**Proposed Solution**
How should it work?

**Alternatives Considered**
Other approaches considered.

**Additional Context**
Any other relevant information.
```

### Asking Questions

For questions:
- Check [existing documentation](docs/)
- Search [existing issues](https://github.com/neogan74/k8s-chaos/issues)
- Use [Discussions](https://github.com/neogan74/k8s-chaos/discussions) for general questions
- Open an issue if it's a potential bug

## Architecture Decision Records

For significant architectural changes:

1. **Propose**: Create ADR with "Proposed" status
2. **Discuss**: Open PR for review and discussion
3. **Decide**: Team reviews and approves/rejects
4. **Implement**: Update status to "Accepted" and implement
5. **Document**: Update architecture documentation

See [docs/adr/README.md](docs/adr/README.md) for details.

## Community

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and discussions
- **Pull Requests**: Code contributions and reviews
- **Documentation**: In-depth guides and references

### Getting Help

- **Documentation**: Check [docs/](docs/) directory
- **Examples**: See [config/samples/](config/samples/)
- **Issues**: Search existing issues
- **Discussions**: Ask in GitHub Discussions

### Recognition

Contributors are recognized through:
- Credit in release notes
- Contributor list in README
- GitHub contributor graphs
- Special mentions for significant contributions

## Development Tips

### Useful Commands

```bash
# Build binary
make build

# Build Docker image
make docker-build IMG=myrepo/k8s-chaos:tag

# Deploy to cluster
make deploy IMG=myrepo/k8s-chaos:tag

# Run locally against cluster
make run

# Debug with delve
dlv debug ./cmd/main.go

# View controller logs
kubectl logs -n k8s-chaos-system deployment/k8s-chaos-controller-manager -f

# Clean up
make dev-clean
```

### Debugging

```bash
# Enable debug logging
make run ARGS="--zap-log-level=debug"

# Use delve debugger
dlv debug ./cmd/main.go -- --metrics-bind-address=:8080

# Check CRD status
kubectl get crds chaosexperiments.chaos.gushchin.dev -o yaml

# Describe experiment
kubectl describe chaosexperiment <name> -n <namespace>
```

### Common Development Tasks

**Adding a New Chaos Action:**

1. Update API types if needed
2. Add action to validation
3. Implement execution logic in controller
4. Add unit tests
5. Create sample CRD
6. Update documentation
7. Create ADR if significant

**Updating CRD Schema:**

```bash
# 1. Edit types
vim api/v1alpha1/chaosexperiment_types.go

# 2. Regenerate manifests
make manifests generate

# 3. Update samples
vim config/samples/

# 4. Update docs
vim docs/API.md
```

## License

By contributing to k8s-chaos, you agree that your contributions will be licensed under the Apache License 2.0.

## Questions?

Don't hesitate to ask! Open an issue or discussion if you need help.

Thank you for contributing to k8s-chaos!