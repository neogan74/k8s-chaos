# Network Partition Implementation Research

**Date**: 2026-02-10
**Status**: Research Complete

## Executive Summary

Research confirms current iptables approach is industry-standard for network partitions in Kubernetes chaos engineering. tc netem better suited for degradation (latency, loss), iptables better for binary isolation. Selective blocking by IP/CIDR/service is achievable enhancement requiring additional API fields and iptables rules with proper chain management.

## Industry Approaches

### Chaos Engineering Tools Comparison

**Chaos Mesh** (most mature):
- Abstracts iptables/tc complexity with declarative APIs
- Uses iptables for partition, tc netem for degradation
- Supports selective targeting by pod/namespace/service
- Source: [Chaos Mesh Network Chaos](https://deepwiki.com/chaos-mesh/chaos-mesh/3.2-network-chaos)

**Harness Chaos Engineering**:
- Separates "network partition" from "network loss" actions
- Network loss uses tc netem for packet loss simulation
- Network partition uses iptables for complete isolation
- Source: [Harness Chaos Faults](https://developer.harness.io/docs/chaos-engineering/faults/chaos-faults/kubernetes/)

**Pumba** (Docker-focused):
- Uses tc for network emulation (latency, loss)
- Uses iptables for packet filtering and partitions
- Supports asymmetric conditions (netem outgoing, iptables incoming)
- Source: [Pumba GitHub](https://github.com/alexei-led/pumba)

**Key Insight**: Industry consensus separates concerns:
- **tc netem**: Probabilistic degradation (latency, jitter, loss percentage)
- **iptables**: Deterministic blocking (complete isolation, selective filtering)

### iptables vs tc netem Decision Matrix

| Criteria | iptables | tc netem |
|----------|----------|----------|
| Use case | Complete partition, selective blocking | Gradual degradation, packet loss |
| Precision | Binary (allow/drop) | Probabilistic (percentage) |
| Direction control | INPUT/OUTPUT chains | egress only (ingress requires tc filters) |
| Cleanup risk | Rule persistence if process dies | qdisc persists if not removed |
| Performance impact | Low (rule evaluation) | Medium (packet processing) |
| Complexity | Moderate (chain management) | Low (simple qdisc commands) |
| Selective targeting | Native (src/dst IP/port) | Limited (requires filters) |
| Best for | Split-brain, partition testing | Flaky network, packet loss |

**Decision**: Current iptables choice correct for network-partition action. tc netem already used for pod-network-loss.

Source: [Traffic Control for Network Chaos](https://songrgg.github.io/operation/use-traffic-control-simulate-network-chaos/)

## Selective Blocking Implementation

### iptables Best Practices

**CIDR Blocking Syntax**:
```bash
# Block specific IP/CIDR
iptables -A INPUT -s 192.168.1.0/24 -j DROP
iptables -A OUTPUT -d 10.96.0.0/12 -j DROP  # Block K8s service CIDR

# Block with comments (for debugging)
iptables -A INPUT -s 192.168.1.100 -m comment --comment "Block backend service" -j DROP

# Block specific ports
iptables -A OUTPUT -d 10.96.100.50 -p tcp --dport 8080 -j DROP
```

Source: [nixCraft iptables IP/CIDR blocking](https://www.cyberciti.biz/tips/linux-iptables-how-to-specify-a-range-of-ip-addresses-or-ports.html)

**Rule Ordering**:
- Order matters: First match wins
- Use `-I` to insert at specific position
- Use `-A` to append to end
- Allow rules before drop rules

Source: [DigitalOcean iptables Essentials](https://www.digitalocean.com/community/tutorials/iptables-essentials-common-firewall-rules-and-commands)

**Chain Management for Safety**:
```bash
# Create custom chain for chaos rules
iptables -N CHAOS_PARTITION
iptables -A INPUT -j CHAOS_PARTITION
iptables -A OUTPUT -j CHAOS_PARTITION

# Add rules to custom chain
iptables -A CHAOS_PARTITION -s 10.96.0.0/12 -j DROP

# Cleanup: Delete chain instead of flush
iptables -D INPUT -j CHAOS_PARTITION
iptables -D OUTPUT -j CHAOS_PARTITION
iptables -F CHAOS_PARTITION
iptables -X CHAOS_PARTITION
```

**Benefits**:
- Isolated from other iptables rules
- Safe cleanup (only removes chaos rules)
- No impact on CNI/service mesh rules
- Easy debugging (inspect chain separately)

Source: [Linode iptables Traffic Control](https://www.linode.com/docs/guides/control-network-traffic-with-iptables/)

**ipset for Large Lists** (future enhancement):
```bash
# Create ipset of blocked IPs
ipset create blocked_ips hash:net
ipset add blocked_ips 192.168.1.0/24
ipset add blocked_ips 10.96.100.0/24

# Single iptables rule referencing set
iptables -A INPUT -m set --match-set blocked_ips src -j DROP
```

**Benefits**: Efficient for blocking many IPs/CIDRs (single rule, O(1) lookup)

Source: [SNBForums ipset best practices](https://www.snbforums.com/threads/best-way-to-automatically-block-add-delete-ip-cidr-ranges.77653/)

## Service-to-Service Partition Scenarios

### Kubernetes Service Resolution

**Challenge**: Users want to specify "block traffic to service X" not "block traffic to IP Y"

**Solution Approach**:
1. API accepts service names (e.g., "redis-service")
2. Controller resolves service to ClusterIP
3. Injects iptables rules blocking that IP
4. Handles service IP changes (watch service resources)

**Example API**:
```yaml
spec:
  action: network-partition
  targetServices:
    - name: redis-service
      namespace: backend
      ports: [6379]
  targetNamespaces:
    - backend
    - database
```

**iptables Translation**:
```bash
# Resolved from service
REDIS_IP=$(kubectl get svc redis-service -n backend -o jsonpath='{.spec.clusterIP}')

# Block traffic to service
iptables -A OUTPUT -d $REDIS_IP -p tcp --dport 6379 -j DROP
```

### Namespace-Level Partition

**Use Case**: Simulate network partition between namespaces (e.g., frontend can't reach backend)

**Implementation**:
1. List all pods in target namespace
2. Extract pod IPs
3. Block traffic to all those IPs
4. Use ipset for efficiency

**Example**:
```bash
# Get all pod IPs in namespace
BACKEND_IPS=$(kubectl get pods -n backend -o jsonpath='{.items[*].status.podIP}')

# Create ipset and block
ipset create backend_pods hash:ip
for ip in $BACKEND_IPS; do
  ipset add backend_pods $ip
done
iptables -A OUTPUT -m set --match-set backend_pods dst -j DROP
```

**Complexity**: Requires pod IP tracking, handles pod churn (new pods added/removed)

## Advanced Features Research

### Protocol/Port Filtering

**Use Case**: Block only specific protocols/ports (e.g., block HTTP but allow HTTPS)

**iptables Support**:
```bash
# Block only HTTP
iptables -A OUTPUT -p tcp --dport 80 -j DROP

# Block UDP DNS
iptables -A OUTPUT -p udp --dport 53 -j DROP

# Block all TCP but allow specific port
iptables -A OUTPUT -p tcp ! --dport 22 -j DROP
```

**API Design**:
```yaml
spec:
  targetProtocols:
    - protocol: tcp
      ports: [80, 8080]
    - protocol: udp
      ports: [53]
```

### Asymmetric Partitions

**Use Case**: Allow traffic in one direction but block the other (realistic split-brain)

**Current Support**: Already have direction field (ingress/egress/both)

**Enhancement**: Combine with selective targeting
```yaml
spec:
  direction: egress  # Can send but not receive
  targetServices: [redis-service]
```

### Network Policy Alternative

**Consideration**: Could use NetworkPolicy instead of iptables

**Pros**:
- Declarative, Kubernetes-native
- No NET_ADMIN capability required
- Works with all CNI plugins

**Cons**:
- Requires network policy support in CNI
- Slower to apply (controller reconciliation)
- Less precise (pod-level, not container-level)
- Harder to inject into existing pods

**Decision**: Keep iptables for precision and immediate effect, consider NetworkPolicy as alternative implementation mode

Source: [Coroot Chaos-Driven Observability](https://coroot.com/blog/engineering/chaos-driven-observability-spotting-network-failures/)

## Security and Safety Considerations

### NET_ADMIN Capability

**Current Approach**: Ephemeral container with NET_ADMIN capability

**Pod Security Standards (PSA)**:
- **Privileged**: Allows NET_ADMIN
- **Baseline**: Allows NET_ADMIN
- **Restricted**: Blocks NET_ADMIN

**Impact**: Won't work in PSA Restricted namespaces

**Mitigation**:
1. Document PSA requirements clearly
2. Provide pre-flight validation (check namespace PSA level)
3. Fail fast with helpful error message
4. Consider alternative implementation (NetworkPolicy) for restricted environments

### Rule Persistence Risk

**Problem**: If cleanup fails, iptables rules persist after pod restart

**Current Cleanup**: `iptables -F` (flushes all rules)

**Risk**: May remove CNI/mesh rules

**Improved Cleanup** (custom chain):
```bash
# Setup
iptables -N CHAOS_PARTITION_$EXPERIMENT_ID
iptables -I INPUT 1 -j CHAOS_PARTITION_$EXPERIMENT_ID
iptables -I OUTPUT 1 -j CHAOS_PARTITION_$EXPERIMENT_ID

# Cleanup (only removes chaos rules)
iptables -D INPUT -j CHAOS_PARTITION_$EXPERIMENT_ID
iptables -D OUTPUT -j CHAOS_PARTITION_$EXPERIMENT_ID
iptables -F CHAOS_PARTITION_$EXPERIMENT_ID
iptables -X CHAOS_PARTITION_$EXPERIMENT_ID
```

**Benefits**:
- Isolated namespace for chaos rules
- Safe cleanup (no impact on other rules)
- Deterministic verification (check chain exists)

## Recommendations

### Phase 1: Documentation & Stability (Immediate)
1. Document current iptables approach in ADR
2. Create sample YAML with common scenarios
3. Improve cleanup to use custom chains
4. Add PSA compatibility validation

### Phase 2: Selective Targeting (Short-term)
1. Add API fields: targetIPs, targetCIDRs, targetPorts
2. Implement iptables rules with specific src/dst filtering
3. Add validation for CIDR format
4. Update E2E tests for selective scenarios

### Phase 3: Service-Aware Partitions (Medium-term)
1. Add API fields: targetServices, targetNamespaces
2. Implement service IP resolution
3. Use ipset for efficient IP list management
4. Handle service IP changes (watch services)

### Phase 4: Advanced Features (Long-term)
1. Protocol/port filtering support
2. NetworkPolicy alternative implementation
3. Automatic PSA detection and mode selection
4. Grafana dashboard for partition visualization

## Conclusion

Current iptables implementation aligns with industry best practices. Recommended enhancements focus on selective targeting capabilities and improved cleanup safety using custom chains. Service-aware partitions enable realistic split-brain scenarios requested by users.

## Sources

- [Chaos Mesh Network Chaos Documentation](https://deepwiki.com/chaos-mesh/chaos-mesh/3.2-network-chaos)
- [Harness Chaos Engineering Faults](https://developer.harness.io/docs/chaos-engineering/faults/chaos-faults/kubernetes/)
- [Harness Pod Network Loss](https://developer.harness.io/docs/chaos-engineering/use-harness-ce/chaos-faults/kubernetes/pod/pod-network-loss/)
- [Pumba Chaos Testing Tool](https://github.com/alexei-led/pumba)
- [Network Failure Testing Guide 2026](https://oneuptime.com/blog/post/2026-01-30-network-failure-testing/view)
- [Coroot Chaos-Driven Observability](https://coroot.com/blog/engineering/chaos-driven-observability-spotting-network-failures/)
- [Traffic Control for Network Chaos](https://songrgg.github.io/operation/use-traffic-control-simulate-network-chaos/)
- [Comparing Chaos Engineering Tools](https://blog.container-solutions.com/comparing-chaos-engineering-tools)
- [nixCraft: Block IP with iptables](https://www.cyberciti.biz/tips/howto-block-ipaddress-with-iptables-firewall.html)
- [nixCraft: iptables IP/CIDR Ranges](https://www.cyberciti.biz/tips/linux-iptables-how-to-specify-a-range-of-ip-addresses-or-ports.html)
- [DigitalOcean: iptables Essentials](https://www.digitalocean.com/community/tutorials/iptables-essentials-common-firewall-rules-and-commands)
- [Linode: Control Traffic with iptables](https://www.linode.com/docs/guides/control-network-traffic-with-iptables/)
- [SNBForums: Block IP/CIDR Ranges](https://www.snbforums.com/threads/best-way-to-automatically-block-add-delete-ip-cidr-ranges.77653/)
