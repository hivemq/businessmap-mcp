# Contributing to Businessmap MCP Server

Thank you for considering contributing to the Businessmap MCP Server! This document provides guidelines and information for contributors.

## 🤝 How to Contribute

### Reporting Issues

- **Bug Reports**: Use GitHub Issues to report bugs
- **Feature Requests**: Propose new features or enhancements
- **Questions**: Ask questions about usage or implementation

### Development Process

1. **Fork** the repository
2. **Clone** your fork locally
3. **Create** a feature branch from `main`
4. **Make** your changes
5. **Test** thoroughly
6. **Commit** with clear messages
7. **Push** to your fork
8. **Open** a Pull Request

## 🛠️ Development Setup

### Prerequisites

- Go 1.25 or later
- Git

### Local Development

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/businessmap-mcp.git
cd businessmap-mcp

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o businessmap-mcp
```

### Environment Setup

```bash
# Copy example environment file
cp .env.example .env

# Edit with your Businessmap credentials for testing
# KANBANIZE_API_KEY=your_api_key
# KANBANIZE_BASE_URL=https://your-subdomain.kanbanize.com
```

## 📝 Code Guidelines

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and concise

### Commit Messages

Use conventional commit format:

```
<type>(<scope>): <description>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions or changes
- `chore`: Maintenance tasks

**Examples:**
```
feat(api): add support for card attachments
fix(client): handle empty response gracefully
docs(readme): update testing instructions
```

### Testing

- Write tests for new functionality
- Ensure existing tests pass
- Test with real Kanbanize API (use test cards)
- Update documentation for new features

### Pull Request Guidelines

1. **Clear Description**: Explain what your PR does and why
2. **Small Scope**: Keep PRs focused on single features/fixes
3. **Documentation**: Update docs for new features
4. **Tests**: Include tests for new functionality
5. **Clean History**: Squash commits if necessary

## 🧪 Testing

### Unit Tests

```bash
go test ./...
```

### Integration Tests

```bash
# Test with real Businessmap API
./tests/test_mcp.sh YOUR_CARD_ID
```


## 📋 Project Structure

```
businessmap-mcp/
├── main.go              # Entry point
├── internal/
│   ├── config/          # Configuration management
│   └── kanbanize/       # API client and types
└── tests/               # Test scripts and fixtures
```

### Adding New Features

1. **API Changes**: Modify `internal/kanbanize/`
2. **Configuration**: Update `internal/config/`
3. **Main Logic**: Modify `main.go`
4. **Tests**: Add to `tests/`
5. **Documentation**: Update README.md

## 🚀 Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release PR
4. Tag release after merge
5. Publish release notes

## 📞 Getting Help

- **GitHub Issues**: For bugs and feature requests
- **Discussions**: For questions and community interaction
- **Code Review**: All PRs receive thorough review

## 🙏 Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes
- Documentation credits

Thank you for contributing to making Businessmap MCP Server better! 🎉