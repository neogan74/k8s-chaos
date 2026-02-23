# k8s-chaos Roadmap

This document outlines the vision and development roadmap for k8s-chaos.

## Vision

**Make chaos engineering accessible, safe, and practical for all Kubernetes users.**

k8s-chaos aims to be the go-to lightweight chaos engineering operator that balances power with simplicity, providing production-ready safety features while remaining easy to learn and use.

---

## Current Status (v0.2.0 - February 2026)

### ✅ Core Features (Implemented)

**Chaos Actions:**
- ✅ Pod chaos: kill, delay, CPU stress, memory stress, failure, network loss
- ✅ Node chaos: drain with auto-uncordon

**Safety & Control:**
- ✅ Dry-run mode
- ✅ Maximum percentage limits
- ✅ Production namespace protection
- ✅ Exclusion labels
- ✅ Experiment duration control
- ✅ Cron-based scheduling
- ✅ Retry logic with backoff strategies

**Observability:**
- ✅ Prometheus metrics
- ✅ Grafana dashboards (3 comprehensive dashboards)
- ✅ Experiment history & audit logging
- ✅ Safety metrics tracking

**Documentation:**
- ✅ Comprehensive user guides (Getting Started, Best Practices, Troubleshooting, Scenarios)
- ✅ API documentation
- ✅ CLI tool with rich commands
- ✅ Hands-on labs infrastructure (8 labs)
- ✅ ADRs for all implemented chaos actions

**Deployment:**
- ✅ Helm chart (production-ready, `charts/k8s-chaos/`)
- ✅ GitOps support: ArgoCD, Flux, Kustomize overlays (`deploy/`)

---

## Roadmap by Quarter

### Q1 2026: Production Hardening

**Goal:** Make k8s-chaos enterprise-ready

#### Completed ✅

**Helm Chart** ✅
- Official Helm chart (`charts/k8s-chaos/`), 50+ parameters, one-command install, cert-manager, ServiceMonitor

**GitOps Support** ✅
- ArgoCD Application/ApplicationSet, Flux HelmRelease/Kustomization, Kustomize env overlays

**Pod Network Loss** ✅
- `pod-network-loss` action with `tc netem`, ephemeral containers, safety wiring, Prometheus metrics

#### In Progress / Remaining 🚧

**Test Coverage** 🧪
- Increase unit test coverage to 80% (currently ~50–60%)
- Integration tests for all chaos actions
- E2E test scenarios (Kind)
- **Impact:** Reliability and confidence for production adoption

**pod-disk-fill Implementation** 💽
- ADR and docs complete; controller/webhook/tests pending
- See `docs/adr/0008-pod-disk-fill-implementation.md` and `docs/BACKLOG.md`
- **Impact:** Completes the core infrastructure chaos action set

**Kubernetes Events** 📢
- Emit events on ChaosExperiment and affected pods/nodes
- Populate `status.affectedPods` list
- **Impact:** Better K8s ecosystem integration and debuggability

**Contributing Guide** 📝
- `CONTRIBUTING.md` with dev setup, how to add actions, PR process
- **Impact:** Opens the door to community contributions

**Performance Optimization** ⚡
- Rate limiting, batch operations, memory/CPU profiling
- **Impact:** Scalability for large clusters

---

### Q2 2026: Feature Expansion

**Goal:** Add advanced chaos capabilities

#### New Chaos Actions

**Network Chaos** 🌐
- `pod-network-corruption`: Packet corruption (follows pod-network-loss pattern)
- `pod-network-partition`: Block traffic between pod groups
- `dns-chaos`: DNS resolution failures
- **Impact:** Critical for testing network resilience

**Infrastructure Chaos** 🏗️
- `pod-disk-fill`: Fill pod disk space (ADR+docs done; controller in Q1 2026 backlog)
- `node-taint`: Add taints to nodes
- `node-disk-fill`: Fill node disk space
- `node-cpu-stress`: Stress node CPU
- **Impact:** Test infrastructure-level failures

**Application Chaos** 💥
- `http-chaos`: HTTP response manipulation
- `pod-restart`: Graceful pod restart
- **Impact:** Application-specific testing

#### Advanced Features

**Time Windows** ⏰
- Define maintenance windows for experiments
- Automatic pause outside windows
- Integration with operational calendars
- **Impact:** Better operational control

**Experiment Orchestration** 🎼
- Chain multiple chaos actions
- Scenario support (predefined experiment sequences)
- Dependency management between experiments
- **Impact:** Complex testing scenarios

---

### Q3 2026: Enterprise Integration

**Goal:** Enterprise features and integrations

#### Integrations

**Observability** 📊
- Prometheus AlertManager integration
- Slack/PagerDuty notifications
- Custom webhook support
- **Impact:** Better incident response

**Service Mesh** 🕸️
- Istio integration for advanced network chaos
- Linkerd support
- Service mesh-aware chaos injection
- **Impact:** Cloud-native architecture support

**CI/CD** 🔄
- Argo Workflows integration
- GitOps support
- Automated chaos in pipelines
- **Impact:** Shift-left chaos testing

#### Security & Compliance

**RBAC Enhancements** 🔒
- Fine-grained permissions by chaos action
- Namespace-scoped roles
- Audit logging improvements
- **Impact:** Enterprise security requirements

**Policy Integration** 📋
- OPA (Open Policy Agent) integration
- Policy-based experiment approval
- Compliance reporting
- **Impact:** Regulatory compliance

---

### Q4 2026: Advanced Capabilities

**Goal:** Intelligent chaos engineering

#### AI/ML Features

**Steady State Detection** 🎯
- Automatic baseline detection
- Anomaly detection during experiments
- Smart rollback on SLO violations
- **Impact:** Self-healing experiments

**Impact Analysis** 📈
- Automatic blast radius calculation
- Resource dependency mapping
- Predictive impact modeling
- **Impact:** Better experiment planning

**Learning Mode** 🧠
- Suggest experiments based on topology
- Learn from past experiments
- Automated experiment optimization
- **Impact:** Intelligent chaos engineering

#### Advanced Orchestration

**Conditional Chaos** 🔀
- Trigger experiments based on metrics/alerts
- Event-driven chaos injection
- Gradual chaos (increase intensity over time)
- **Impact:** Dynamic testing

**Multi-tenancy** 👥
- Support for multiple teams
- Quota management per team
- Isolated experiment namespaces
- **Impact:** Large organization support

---

## Beyond 2026: Future Vision

### Web UI/Dashboard 🖥️
- Visual experiment designer
- Real-time monitoring dashboard
- Experiment catalog and templates
- Historical analysis and reporting

### Multi-Cluster Support 🌍
- Coordinate chaos across clusters
- Cross-cluster dependency testing
- Regional failure simulation

### Chaos-as-a-Service ☁️
- Managed chaos engineering platform
- Pre-built experiment libraries
- Industry-specific scenarios
- SaaS offering

### Community Ecosystem 🌱
- Plugin system for custom actions
- Marketplace for experiments
- Integration library
- Conference talks and workshops

---

## How to Contribute

We welcome contributions in these areas:

### Immediate Needs (Q1 2026)
1. **pod-disk-fill** - Implement controller, webhook validation, and tests (ADR 0008 ready)
2. **Test Coverage** - Write unit and integration tests to reach 80%
3. **Kubernetes Events** - Emit events on experiments and affected pods
4. **Contributing Guide** - Help new contributors get started
5. **Bug Fixes** - Address issues as they arise

### Medium Term (Q2-Q3 2026)
1. **New Chaos Actions** - Implement network/infrastructure chaos
2. **Integrations** - Build service mesh/observability integrations
3. **CLI Enhancements** - Add interactive wizards and validation

### Long Term (Q4 2026+)
1. **ML Features** - Contribute to intelligent capabilities
2. **Web UI** - Build visual dashboard
3. **Multi-cluster** - Design and implement cross-cluster support

### How to Get Started

1. **Pick an Issue**: Check [GitHub Issues](https://github.com/neogan74/k8s-chaos/issues)
2. **Discuss First**: Open a discussion for large features
3. **Follow Guidelines**: Read `CONTRIBUTING.md` (coming soon!)
4. **Submit PR**: Follow our PR template and code review process

---

## Priority Framework

We prioritize work based on:

### 🔴 Critical (P0)
- Blocks basic functionality
- Security vulnerabilities
- Data loss risks
- Production incidents

### 🟡 High (P1)
- Major features from roadmap
- Performance issues
- Important integrations
- User-requested features with broad impact

### 🟢 Medium (P2)
- Nice-to-have features
- Documentation improvements
- Code quality enhancements
- Minor bug fixes

### 🔵 Low (P3)
- Future enhancements
- Experimental features
- Long-term improvements
- Research projects

---

## Success Metrics

We measure success by:

**Adoption** 📊
- GitHub stars
- Docker pulls
- Active installations
- Community size

**Quality** ✅
- Test coverage (target: 80%)
- Bug report response time
- Issue resolution rate
- User satisfaction

**Community** 👥
- Contributors
- PR submissions
- Discussions/questions
- Conference talks

**Impact** 🎯
- Production deployments
- Enterprise adoption
- Case studies
- Success stories

---

## Feedback & Suggestions

This roadmap is a living document. We value community input!

- **GitHub Discussions**: Share ideas and feedback
- **GitHub Issues**: Request specific features
- **Email**: Contact maintainers directly
- **Community Calls**: Monthly roadmap review (coming soon!)

---

## Release Cadence

**Major Releases** (X.0.0): Quarterly
- Significant new features
- Breaking changes (if necessary)
- Major improvements

**Minor Releases** (0.X.0): Monthly
- New features
- Enhancements
- Non-breaking changes

**Patch Releases** (0.0.X): As needed
- Bug fixes
- Security patches
- Critical fixes

---

## Stay Updated

- **GitHub**: Watch the repository for updates
- **Releases**: Subscribe to release notifications
- **Blog**: Read our blog for detailed updates (coming soon!)
- **Twitter**: Follow [@k8schaos](https://twitter.com/k8schaos) (coming soon!)

---

*Last Updated: February 23, 2026*
*Next Review: April 1, 2026*

**Questions?** Open a [GitHub Discussion](https://github.com/neogan74/k8s-chaos/discussions)