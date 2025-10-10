# k8s-chaos Documentation

Welcome to the k8s-chaos documentation! This directory contains comprehensive guides for developers and users.

## 📚 Documentation Index

### For Developers
- **[Quick Start](QUICKSTART.md)** - Get running in 5 minutes
- **[Development Guide](DEVELOPMENT.md)** - Complete local development setup
- **[Architecture Decision Records](adr/README.md)** - Design decisions and rationale
- **[CLAUDE.md](../CLAUDE.md)** - Guidance for Claude Code AI assistant

### For Users
- **[API Reference](API.md)** - Complete CRD field documentation
- **[Sample CRDs](../config/samples/README.md)** - Example chaos experiments
- **[Project README](../Readme.md)** - Project overview and installation

## 🚀 Quick Commands

### First Time Setup
```bash
# Complete development environment
make dev-setup

# Run controller locally
make dev-run

# Start chaos experiments
make demo-run
```

### Daily Development
```bash
# Check environment status
make dev-status

# Run tests
make test lint

# Reset demo environment
make demo-reset
```

### Cleanup
```bash
# Remove everything
make dev-clean
```

## 📋 Make Commands Reference

### Local Development
| Command | Description |
|---------|-------------|
| `make dev-setup` | Complete development environment setup |
| `make dev-dependencies` | Install development tools |
| `make dev-cluster` | Create Kind cluster |
| `make dev-install` | Install CRDs to cluster |
| `make dev-demo` | Deploy demo environment |
| `make dev-run` | Run controller locally |
| `make dev-status` | Show environment status |
| `make dev-clean` | Clean up everything |

### Demo Operations
| Command | Description |
|---------|-------------|
| `make demo-run` | Start chaos experiment |
| `make demo-watch` | Watch pods in real-time |
| `make demo-status` | Show detailed status |
| `make demo-stop` | Stop all experiments |
| `make demo-reset` | Reset demo to clean state |

### Standard Operations
| Command | Description |
|---------|-------------|
| `make build` | Build manager binary |
| `make test` | Run unit tests |
| `make lint` | Run linter |
| `make manifests` | Generate CRDs and RBAC |
| `make install` | Install CRDs to cluster |
| `make deploy` | Deploy controller to cluster |

## 🎯 Common Workflows

### Adding New Features
1. Read [Development Guide](DEVELOPMENT.md)
2. Set up environment: `make dev-setup`
3. Make changes to code
4. Test: `make test lint`
5. Test locally: `make dev-run`
6. Create samples and documentation

### Testing Changes
1. Start controller: `make dev-run`
2. Run experiments: `make demo-run`
3. Watch results: `make demo-watch`
4. Check status: `make demo-status`
5. Stop when done: `make demo-stop`

### Contributing
1. Fork the repository
2. Follow [Development Guide](DEVELOPMENT.md)
3. Add tests for new features
4. Update documentation
5. Submit pull request

## 🛡️ Safety Guidelines

### Development Safety
- Always test in isolated environments
- Use demo namespace for experiments
- Start with small pod counts
- Monitor cluster resources

### Production Considerations
- Test thoroughly in staging first
- Implement proper RBAC
- Monitor experiment impact
- Have rollback procedures

## 🔧 Troubleshooting

### Common Issues
- **Docker not running**: Start Docker Desktop/daemon
- **Kind cluster issues**: Run `make dev-clean` and retry
- **Permission errors**: Check kubectl context and RBAC
- **CRD not found**: Run `make dev-install`

### Getting Help
1. Check existing documentation
2. Look at sample configurations
3. Review controller logs
4. Check GitHub issues

## 📖 Additional Resources

- **[Kubebuilder Book](https://book.kubebuilder.io/)** - Controller development
- **[Kind Documentation](https://kind.sigs.k8s.io/)** - Local Kubernetes
- **[Chaos Engineering Principles](https://principlesofchaos.org/)** - Theory and best practices

---

Happy chaos engineering! 😈