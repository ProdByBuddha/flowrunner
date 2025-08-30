# Docker Configuration

Template for Docker container and image management

## Environment Variables

```bash
export DOCKER_HOST="your_value_here"
export DOCKER_ALLOW_RUN="your_value_here"
export DOCKER_ALLOWED_IMAGES="your_value_here"
export DEBUG_DOCKER="your_value_here"
```

## Usage

```bash
node examples/mcp-multi-host.js --config examples/config-templates/docker-service.json
```
