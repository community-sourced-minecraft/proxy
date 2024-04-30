# Proxy

## Developing

### Prerequisites

-   [Docker](https://www.docker.com/)
-   [Docker Compose](https://docs.docker.com/compose/)
-   [Go](https://golang.org/)

### Running

```bash
docker compose -f compose.dev.yml up -d

# Apply the development configuration that Gate uses
cp config.dev.yml config.yml

go run .
```
