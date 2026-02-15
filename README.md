# IPWhere

[![Build and Publish Docker Image](https://github.com/jcjc-dev/ipwhere/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/jcjc-dev/ipwhere/actions/workflows/docker-publish.yml)

An all-in-one IP geolocation lookup server inspired by [echoip](https://github.com/mpolden/echoip). This project provides a simple, self-hosted solution for looking up IP address information including country, city, coordinates, timezone, and ASN data.

**🌐 https://ip.shoyu.dev**

## Features

- 🌍 **IP Geolocation**: Look up country, city, region, coordinates, and timezone for any IP address
- 🗺️ **Interactive Map**: View IP locations on an OpenStreetMap-based map
- 🏢 **ASN Information**: Get Autonomous System Number and organization details
- 🔌 **RESTful API**: Clean JSON API with selective field returns
- 📖 **OpenAPI/Swagger**: Auto-generated API documentation
- 🎨 **Modern Web UI**: Beautiful frontend built with TypeScript and Tailwind CSS
- 🐳 **Docker Ready**: Multi-arch container images (amd64/arm64)
- 🔒 **Headless Mode**: Run as API-only server without frontend

## Data Source

This project uses the [DB-IP Lite](https://db-ip.com/db/lite.php) databases, which are less restrictive than alternatives and can be used for any scenarios you see fit. The MMDB database files are fetched from [jcjc-dev/mmdb-latest](https://github.com/jcjc-dev/mmdb-latest) which provides daily-updated releases.

### Attribution

<a href='https://db-ip.com'>IP Geolocation by DB-IP</a>

The DB-IP Lite databases are licensed under [Creative Commons Attribution 4.0 International License](https://creativecommons.org/licenses/by/4.0/). As required by this license, proper attribution is included in:
- All API responses (via `attribution` field)
- The web frontend footer

## Quick Start

### Using Docker (Recommended)

#### Server Mode (Web UI + API)

```bash
# Run with frontend enabled (default)
docker run -p 8080:8080 ghcr.io/jcjc-dev/ipwhere:latest

# Run in headless mode (API only)
docker run -p 8080:8080 -e HEADLESS=true ghcr.io/jcjc-dev/ipwhere:latest
```

Then open http://localhost:8080 in your browser or use the API.

#### CLI Mode (Direct Lookup)

For quick one-off lookups without starting a server:

```bash
# Look up an IP address directly
docker run --rm ghcr.io/jcjc-dev/ipwhere 8.8.8.8
```

Output:
```json
{
  "ip": "8.8.8.8",
  "hostname": "dns.google",
  "country": "United States",
  "iso_code": "US",
  "city": "Mountain View",
  "region": "California",
  "latitude": 37.422,
  "longitude": -122.085,
  "asn": 15169,
  "organization": "Google LLC",
  "attribution": "IP Geolocation by DB-IP (https://db-ip.com)"
}
```

You can also create a shell alias for convenience:
```bash
alias ipwhere='docker run --rm ghcr.io/jcjc-dev/ipwhere'
ipwhere 1.1.1.1
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jcjc-dev/ipwhere.git
cd ipwhere

# Build and run with Docker
docker build -t ipwhere .
docker run -p 8080:8080 ipwhere

# Or build multi-arch images
docker buildx build --platform linux/amd64,linux/arm64 -t ipwhere .
```

## API Usage

> 💡 **Tip:** Try these API calls at https://ip.shoyu.dev — just replace `localhost:8080` with `ip.shoyu.dev`. Excessive usage may be blocked.

### Get Your IP Information

```bash
# Get full IP information as JSON
curl http://localhost:8080/api/ip

# Get information for a specific IP
curl http://localhost:8080/api/ip?ip=8.8.8.8

# Get specific fields only
curl "http://localhost:8080/api/ip?return=country&return=city"

# Get single field
curl "http://localhost:8080/api/ip?return=country"
```

### API Response Example

```json
{
  "ip": "8.8.8.8",
  "country": "United States",
  "iso_code": "US",
  "in_eu": false,
  "city": "Mountain View",
  "region": "California",
  "latitude": 37.4056,
  "longitude": -122.0775,
  "timezone": "America/Los_Angeles",
  "asn": 15169,
  "organization": "Google LLC",
  "attribution": "IP Geolocation by DB-IP (https://db-ip.com)"
}
```

### Available Fields

| Field | Description |
|-------|-------------|
| `ip` | The queried IP address |
| `country` | Country name |
| `iso_code` | ISO 3166-1 alpha-2 country code |
| `in_eu` | Whether the country is in the European Union |
| `city` | City name |
| `region` | Region/State name |
| `latitude` | Latitude coordinate |
| `longitude` | Longitude coordinate |
| `timezone` | IANA timezone identifier |
| `asn` | Autonomous System Number |
| `organization` | AS organization name |

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/ip` | Get IP information for the requesting client |
| `GET /api/ip?ip=x.x.x.x` | Get IP information for a specific IP |
| `GET /api/ip?return=field` | Return only specific fields (repeatable) |
| `GET /swagger/` | OpenAPI/Swagger documentation |
| `GET /health` | Health check endpoint |

## Configuration

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-l, --listen` | Address to listen on | `:8080` |
| `-H, --headless` | Disable frontend, API only | `false` |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LISTEN_ADDR` | Address to listen on | `:8080` |
| `HEADLESS` | Set to `true` to disable frontend | `false` |

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- npm 10+

### Local Development

```bash
# Install Go dependencies
go mod download

# Install frontend dependencies
cd web && npm install && cd ..

# Run frontend dev server (with hot reload)
cd web && npm run dev

# Run Go server (in another terminal)
go run ./cmd/ipwhere

# Run tests
go test ./...
cd web && npm test
```

### Building

```bash
# Build frontend
cd web && npm run build && cd ..

# Build Go binary (frontend must be built first)
go build -o ipwhere ./cmd/ipwhere

# Generate Swagger docs
swag init -g cmd/ipwhere/main.go
```

## License

This project's source code is licensed under the [MIT License](LICENSE).

**Important**: The DB-IP Lite database files are licensed separately under [Creative Commons Attribution 4.0 International License](https://creativecommons.org/licenses/by/4.0/) and are NOT covered by the MIT license. When using this project, you must comply with the DB-IP license terms, including proper attribution.

## Acknowledgments

- [echoip](https://github.com/mpolden/echoip) - Inspiration for this project
- [DB-IP](https://db-ip.com) - IP geolocation database provider
- [jcjc-dev/mmdb-latest](https://github.com/jcjc-dev/mmdb-latest) - MMDB database releases
