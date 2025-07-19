# Contributors

This document acknowledges the individuals and organizations who have contributed to the FlowRunner project.

## Core Development Team

### Lead Developer
- **Trevor Martin** ([@tcmartin](https://github.com/tcmartin))
  - Project founder and lead architect
  - Core infrastructure design and implementation
  - Storage layer architecture and multi-backend support
  - Security and authentication systems

## Contribution Areas

### Architecture & Core Systems
- **Trevor Martin** - Project architecture, storage abstraction, security framework

### Storage Backends
- **Trevor Martin** - DynamoDB provider, PostgreSQL provider, in-memory provider

### Node Implementations
- **Trevor Martin** - HTTP, Email, LLM, Storage, AI Agent, Scheduling nodes

### Testing & Quality Assurance
- **Trevor Martin** - Test framework design, integration testing, performance testing

### Documentation
- **Trevor Martin** - Technical documentation, API documentation, implementation guides

## How to Contribute

We welcome contributions from the community! Here are some ways you can help:

### Code Contributions
- Implement new node types
- Improve existing functionality
- Add test coverage
- Fix bugs and issues

### Documentation
- Improve existing documentation
- Add examples and tutorials
- Create user guides
- Translate documentation

### Testing
- Report bugs and issues
- Test new features
- Performance testing
- Security testing

### Community Support
- Help answer questions in discussions
- Review pull requests
- Share your use cases and examples

## Contribution Guidelines

### Before Contributing
1. Check existing issues and pull requests
2. Read the [development guidelines](INTERNAL_PROGRESS.md#development-guidelines)
3. Set up your development environment
4. Run the test suite to ensure everything works

### Submitting Contributions
1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes with appropriate tests
4. Ensure all tests pass
5. Submit a pull request with a clear description

### Code Style
- Follow Go best practices and conventions
- Write comprehensive tests for new features
- Document public APIs and complex logic
- Use meaningful commit messages

### Testing Requirements
- Maintain minimum 80% test coverage
- Include unit tests for new functionality
- Add integration tests for cross-component features
- Performance test critical paths

## Recognition

We recognize contributions in several ways:
- Listed in this CONTRIBUTORS.md file
- Mentioned in release notes for significant contributions
- GitHub contributor statistics
- Special recognition for outstanding contributions

## Communication

### Getting Help
- GitHub Issues for bugs and feature requests
- GitHub Discussions for questions and community support
- Email: [project-email] for private matters

### Development Coordination
- GitHub Projects for tracking development progress
- Regular contributor meetings (as the project grows)
- Development updates in GitHub Discussions

## License

By contributing to FlowRunner, you agree that your contributions will be licensed under the MIT License.

## Acknowledgments

### Special Thanks
- The [FlowLib](https://github.com/tcmartin/flowlib) project for providing the underlying flow execution engine
- The Go community for excellent libraries and tools
- AWS, PostgreSQL, and other service providers for robust infrastructure

### Inspiration
FlowRunner was inspired by the need for a lightweight, YAML-driven workflow orchestration system that could bridge the gap between simple automation scripts and complex enterprise workflow engines.

---

**Note**: This contributor list is maintained manually. If you've contributed and don't see your name, please open an issue or submit a pull request to add yourself.

Last updated: July 19, 2025
