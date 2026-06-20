# ADR 0011: Network Partition Implementation

**Status:** Implemented

**Date:** 2026-02-11

**Authors:** k8s-chaos team

## Context

Network partitions are critical failure scenarios in distributed systems, causing split-brain conditions, data inconsistency, and availability issues. Testing resilience to network partitions is essential for validating distributed system behavior, consensus protocols, leader election, and quorum mechanisms.

We need a Kubernetes-native way to simulate complete network isolation between pods and external systems. Unlike partial degradation (latency, packet loss), partitions simulate total connectivity failure—pods become unreachable to prevent or receive network traffic.

**Requirements:**
- Complete network isolation (not just degradation)
- Support directional control (ingress, egress, or both)
- Safe cleanup that doesn't affect CNI or service mesh rules
- Reuse existing ChaosExperiment patterns (safety limits, dry-run, duration)
- No cluster-wide privileges or node agents required
- Works with standard Kubernetes networking (no CNI-specific implementation)

**Constraints:**
- Must honor safety features: dry-run, maxPercentage, exclusions, production protection
- Must integrate with metrics and history logging
- Must be deterministic and reversible
- Cannot require privileged host access

## Decision

Implement `pod-network-partition` using Linux `iptables` with **custom chains** applied from ephemeral containers injected into target pods. Ephemeral containers share the pod network namespace, allowing complete network isolation without sidecars or host access.

### Key Implementation Details

**Chaos Action**: `network-partition`

**Mechanism**: Ephemeral container with `NET_ADMIN` capability running `iptables` commands to:
1. Create a custom iptables chain (`CHAOS_PARTITION_<timestamp>`)
2. Insert jump rules to custom chain in INPUT/OUTPUT chains
3. Add rules to custom chain: allow loopback, drop other traffic
4. Wait for specified duration
5. Clean up by removing only the custom chain (safe, isolated)

**Custom Chain Pattern**:
```bash
# Create custom chain with unique name
iptables -N CHAOS_PARTITION_1707654321

# Insert jump rules (high priority)
iptables -I INPUT 1 -j CHAOS_PARTITION_1707654321
iptables -I OUTPUT 1 -j CHAOS_PARTITION_1707654321

# Rules in custom chain
iptables -A CHAOS_PARTITION_1707654321 -i lo -j ACCEPT  # Allow loopback
iptables -A CHAOS_PARTITION_1707654321 -o lo -j ACCEPT
iptables -A CHAOS_PARTITION_1707654321 -j DROP          # Drop everything else

# Cleanup (removes only chaos rules, safe)
iptables -D INPUT -j CHAOS_PARTITION_1707654321
iptables -D OUTPUT -j CHAOS_PARTITION_1707654321
iptables -F CHAOS_PARTITION_1707654321
iptables -X CHAOS_PARTITION_1707654321
```

**Why Custom Chains?**
- Isolates chaos rules from CNI/mesh networking rules
- Safe cleanup—only removes chaos chain, not system rules
- Prevents conflicts with Calico, Cilium, Istio, Linkerd
- Easy debugging—inspect chain separately
- Deterministic verification—check chain exists/removed

**Lifecycle**:
- Start: Inject ephemeral container with iptables script
- Duration: Script sleeps for specified duration
- Cleanup: Script removes custom chain before exiting
- Tracking: Store affected pods in status for history/metrics

**Spec Fields** (validated by CRD + webhook):
- `action: pod-network-partition` (required)
- `duration` (required) - how long to maintain partition
- `direction` (enum: `both` | `ingress` | `egress`; default: `both`)
- `targets` use existing selector/count/safety filters
- All safety features: `dryRun`, `maxPercentage`, exclusion labels, `allowProduction`

**Safety**:
- Respects maxPercentage to prevent over-affecting resources
- Honors exclusion labels (`chaos.gushchin.dev/exclude: "true"`)
- Requires allowProduction=true for production namespaces
- Dry-run mode previews affected pods without execution
- Gracefully handles terminating pods (auto-excluded)

**Observability**:
- Prometheus metrics: experiments_total, duration, resources_affected, errors
- ChaosExperimentHistory records with affected resources
- Kubernetes events on affected pods
- Status tracking: lastRunTime, message, phase

**Image**: `nicolaka/netshoot` - public image with iptables, widely used in Kubernetes debugging

## Alternatives Considered

### Alternative 1: tc netem (traffic control with network emulation)

**Description**: Use `tc qdisc add dev eth0 root netem loss 100%` to simulate 100% packet loss.

**Pros**:
- Uses same mechanism as pod-network-loss action
- Familiar to ops teams using tc for traffic shaping

**Cons**:
- Semantically incorrect—100% loss ≠ partition (TCP retries, timeouts differ)
- Less precise than complete DROP at firewall layer
- Harder to express directional control
- Performance overhead from netem queueing

**Why Rejected**: iptables DROP is semantically correct for partition simulation and more efficient.

### Alternative 2: NetworkPolicy-based isolation

**Description**: Create Kubernetes NetworkPolicy to deny all ingress/egress.

**Pros**:
- No privileged capabilities required
- Native Kubernetes resource
- Works across CNI plugins

**Cons**:
- Requires CNI plugin support (not guaranteed in all clusters)
- NetworkPolicy application is asynchronous—no deterministic timing
- Cannot guarantee cleanup on experiment failure
- Less precise control (can't do loopback exemption easily)
- Harder to implement duration-based automatic removal

**Why Rejected**: Less reliable and harder to coordinate cleanup. Could be added as alternative implementation in future.

### Alternative 3: Service Mesh Fault Injection (Istio/Linkerd)

**Description**: Use service mesh features to inject network faults.

**Pros**:
- No pod-level privileges needed
- Rich policy expression
- Integrates with existing mesh monitoring

**Cons**:
- Only works in meshed environments (not universal)
- Configuration varies per mesh (Istio vs Linkerd)
- Requires mesh installation/configuration overhead
- Doesn't work for non-meshed traffic (node-to-pod, pod-to-external)

**Why Rejected**: Too environment-specific. Could be added as mesh-aware action in future.

### Alternative 4: iptables without custom chains (original implementation)

**Description**: Use `iptables -A INPUT -j DROP` directly and `iptables -F` for cleanup.

**Pros**:
- Simpler implementation (fewer commands)
- Same result during experiment execution

**Cons**:
- **DANGEROUS**: `iptables -F` flushes ALL rules in filter table
- Removes CNI plugin rules (Calico, Cilium, Weave)
- Removes service mesh rules (Istio, Linkerd)
- Can permanently break pod networking
- No isolation for debugging

**Why Rejected**: Unsafe cleanup creates high risk of operational incidents. Custom chains solve this.

## Consequences

### Positive

- Enables realistic split-brain and network isolation testing
- Safe cleanup prevents CNI/mesh rule conflicts
- Reuses existing ephemeral container injection pipeline
- Custom chains provide clear isolation and debuggability
- Deterministic cleanup reduces risk of lingering network issues
- Works with any CNI plugin (CNI-agnostic)
- No cluster-wide privileges required
- Integrates seamlessly with existing safety features

### Negative

- Requires `NET_ADMIN` capability—blocked in PSA Restricted namespaces
- Adds dependency on nicolaka/netshoot image (28MB)
- Custom chain name must be unique (uses timestamp)
- Ephemeral containers remain in pod spec after experiment (Kubernetes limitation)
- Limited to pod-level network isolation (cannot partition entire namespaces directly)

### Neutral

- Adds new validation paths in webhook for duration requirement
- Increases E2E test surface (custom chain verification)
- Documentation must clarify difference from pod-network-loss
- Users must ensure cluster allows NET_ADMIN in target namespaces

## API Reference

### Selective Targeting Fields (Phase 2)

Network-partition supports selective targeting to block specific traffic instead of complete isolation:

**TargetIPs** (optional):
- Type: `[]string`
- Description: Exact IP addresses to block
- Examples: `["10.96.0.50", "192.168.1.100"]`
- Validation: Must be valid IPv4 addresses
- Warnings: System warns if targeting loopback, cluster IPs, or DNS

**TargetCIDRs** (optional):
- Type: `[]string`
- Description: IP ranges in CIDR notation to block
- Examples: `["10.96.0.0/12", "192.168.0.0/16"]`
- Format: `x.x.x.x/y` where x is 0-255, y is 0-32
- Validation: Must be valid CIDR notation, IPv4 only
- Warnings: System warns if overlapping with cluster service ranges

**TargetPorts** (optional):
- Type: `[]int32`
- Description: Port numbers to block
- Examples: `[80, 443, 8080]`
- Range: 1-65535
- Default protocol: TCP (if targetProtocols not specified)

**TargetProtocols** (optional):
- Type: `[]string`
- Description: Protocols to block
- Valid values: `tcp`, `udp`, `icmp`
- Validation: Enum validation via OpenAPI
- Used with: targetPorts for protocol-specific blocking

**Backward Compatibility**:
- All fields optional
- Empty targets = full partition (existing behavior)
- Direction field applies to selective targets

**Examples**:
```yaml
# Example 1: Block specific IP
targetIPs: ["10.96.100.50"]

# Example 2: Block IP range
targetCIDRs: ["10.96.0.0/12"]

# Example 3: Block HTTP traffic on all IPs
targetPorts: [80, 8080]
targetProtocols: ["tcp"]

# Example 4: Combined - block HTTPS to specific service
targetIPs: ["10.96.100.50"]
targetPorts: [443]
targetProtocols: ["tcp"]
direction: "egress"
```

## Implementation Status

### Completed

- [x] Core controller logic (handleNetworkPartition, injectNetworkPartitionContainer)
- [x] CRD enum value for pod-network-partition action
- [x] Webhook validation (duration required, direction validation)
- [x] Integration with safety features (dry-run, maxPercentage, exclusions)
- [x] Prometheus metrics integration
- [x] History recording via ChaosExperimentHistory CRD
- [x] E2E tests in test/e2e/network_partition_test.go
- [x] Basic implementation with ephemeral containers

### Phase 1: Documentation & Stability (Completed 2026-02-12)

- [x] ADR 0011 documentation (this document)
- [x] Sample YAML in config/samples/chaos_v1alpha1_chaosexperiment_network_partition.yaml
- [x] Custom chain implementation (replace iptables -F)
- [x] CLAUDE.md documentation updates
- [x] Enhanced E2E tests for custom chain verification

### Phase 2: API Design for Selective Targeting (Completed 2026-02-16)

- [x] API fields: targetIPs, targetCIDRs, targetPorts, targetProtocols
- [x] Validation helpers: ValidateCIDR, ValidateIP, ValidatePortRange
- [x] Dangerous target detection: IsDangerousTarget, IsDangerousCIDR
- [x] Webhook validation integration
- [x] Comprehensive unit tests (67 test cases)
- [x] CRD regeneration with new fields

### Planned (Phase 3-4)

- [ ] Controller implementation for selective targeting
- [ ] Service-aware targeting: targetServices, targetNamespaces
- [ ] ipset integration for efficient large-scale targeting
- [ ] PSA compatibility validation with clear error messages
- [ ] E2E tests for selective targeting scenarios

### Deferred

- [ ] NetworkPolicy-based alternative implementation
- [ ] Service mesh integration (Istio/Linkerd)
- [ ] eBPF-based advanced filtering
- [ ] Auto-uncordon for node-partition equivalent

## References

- Current implementation: `internal/controller/chaosexperiment_controller.go` (lines 2587-2766)
- E2E tests: `test/e2e/network_partition_test.go`
- Similar actions: ADR 0007 (pod-network-loss), ADR 0005 (pod-cpu-stress ephemeral containers)
- Phase 1 plan: `plans/20260210-network-partition-implementation/phase-01-documentation-stability.md`
- Research findings: `plans/20260210-network-partition-implementation/reports/02-research-findings.md`
- Kubernetes ephemeral containers: https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/
- iptables custom chains: https://www.netfilter.org/documentation/HOWTO/packet-filtering-HOWTO-7.html
- nicolaka/netshoot: https://github.com/nicolaka/netshoot

## Notes

**Security Considerations**:
- NET_ADMIN capability is required—document in samples
- PSA Restricted policy blocks NET_ADMIN—add pre-flight validation in future phases
- Custom chain isolation reduces attack surface vs full iptables access

**Operational Considerations**:
- Custom chain names use Unix timestamp for uniqueness
- Ephemeral containers remain in pod spec after experiment (Kubernetes behavior)
- Cleanup script uses `|| true` for idempotency on retries
- Script allows loopback (-i lo, -o lo) to prevent process lockup

**Differences from pod-network-loss**:
- **pod-network-loss**: Partial degradation (5-40% packet loss) using tc netem
- **pod-network-partition**: Complete isolation (100% dropped) using iptables
- Network partition is for testing hard failures, network-loss is for testing degradation

**Future Enhancements**:
Custom chains enable Phase 2-4 features:
- Selective blocking: Add rules for specific IPs/CIDRs instead of DROP all
- Service targeting: Resolve service IPs and add to chain rules
- Port filtering: Block only specific ports/protocols
- ipset integration: Efficiently manage large IP lists
