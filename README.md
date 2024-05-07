# Proxy

## Developing

### Prerequisites

- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Go](https://golang.org/)

### Running

```bash
docker compose -f compose.dev.yml up -d

# Apply the development configuration that Gate uses
cp config.dev.yml config.yml

go run .
```

## Thanks

<div style="display: flex; flex-direction: column; width: fit-content; align-items: center">
  <a href="https://qaze.app">
    <img src="./.misc/qaze.svg" alt="Qaze - The NATS GUI" width="200"/>
  </a>
  <p>Qaze - providing us with <b>the</b> GUI for NATS</p>
</div>
