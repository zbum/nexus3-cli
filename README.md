# nexus3-cli

A command-line tool for managing Docker images stored in **Sonatype Nexus Repository OSS 3.68.1-02**, built on the **Nexus REST API** (`/service/rest/v1/...`).

Inspired by [mlabouardy/nexus-cli](https://github.com/mlabouardy/nexus-cli), but implemented against the Nexus REST API instead of the Docker Registry v2 API — so it works against the regular Nexus port (e.g. `8081`) without requiring a Docker HTTP/HTTPS connector.

---

## Features

- List all Docker images in a repository (`image ls`)
- List tags for a specific image (`image tags`)
- Show manifest details, layers, size, architecture, and creation time (`image info`)
- Delete a specific tag, or keep only the N most recent tags (`image delete`)
- Report per-tag and total image size (`image size`)
- Credentials and host stored in `~/.nexus-cli` (mode `0600`)

---

## Installation

### Homebrew (macOS, Linux)

```bash
brew tap zbum/tap
brew install nexus3-cli
```

Upgrade later with `brew update && brew upgrade nexus3-cli`.

### From source

```bash
git clone https://github.com/zbum/nexus3-cli.git
cd nexus3-cli
make build            # output: dist/nexus3-cli
```

### Cross-compile release binaries

```bash
make build-all        # raw binaries
make release          # also produces .tar.gz archives ready to upload
make checksums        # print SHA-256 of the tarballs (used by the Homebrew formula)
```

### Install to `$GOBIN`

```bash
go install github.com/zbum/nexus3-cli@latest
```

---

## Configuration

Run the interactive configure command once:

```bash
nexus3-cli configure
```

You will be prompted for:

| Field | Example | Notes |
|---|---|---|
| Nexus Host | `http://nexus.example.com:8081` | Main Nexus URL (the same URL you open in a browser) |
| Nexus Username | `admin` | Must have read/delete permissions on the repository |
| Nexus Password | `********` | Stored in plain text in `~/.nexus-cli` — keep file permissions tight |
| Nexus Docker Repository name | `docker-hosted` | Name of the Docker repository (hosted or group), scopes all queries |

Config is saved to `~/.nexus-cli` as `key = "value"` lines. Override the path with `NEXUS_CLI_CONFIG=/path/to/file`.

> **Uses the Nexus REST API, not the Docker Registry v2 API.** `nexus_host` is the ordinary Nexus base URL (e.g. `http://nexus:8081`). No Docker HTTP/HTTPS connector is needed — the CLI talks to `/service/rest/v1/search`, `/v1/components`, and `/v1/components/{id}` under Basic auth.

---

## Usage

### List images

```bash
nexus3-cli image ls
```

### List tags for an image

```bash
nexus3-cli image tags --name my-app
```

### Show image details

```bash
nexus3-cli image info --name my-app --tag 1.2.3
```

Output includes the Nexus component ID, repository, total size, and a per-asset breakdown (path, size, content type, last modified).

### Delete a specific tag

```bash
nexus3-cli image delete --name my-app --tag 1.2.3
```

Looks the tag up via `GET /service/rest/v1/search?repository=…&format=docker&name=…&version=…`, then calls `DELETE /service/rest/v1/components/{id}`. Deleting the component removes all of its assets (manifest + blobs) in one request.

### Keep only the N most recent tags

```bash
nexus3-cli image delete --name my-app --keep 5
```

Tags are ordered by **blob upload time** (oldest first, with natural sort as tie-breaker), and everything older than the last `N` is removed.

Add `--yes` / `-y` to skip the confirmation prompt in CI pipelines.

### Protect recently uploaded tags

```bash
nexus3-cli image delete --name my-app --keep 5 --keep-within 30d
```

When `--keep-within` is specified alongside `--keep`, tags uploaded within the given duration are never deleted — even if they exceed the `--keep` count. This means the final number of retained tags may be greater than `N`.

Supported duration formats: `30d` (days), `720h` (hours), `48h30m`, or any Go `time.ParseDuration` syntax.

### Show image size

```bash
nexus3-cli image size --name my-app
```

Prints a per-tag breakdown and a total. Note the total counts shared layers once per tag, so it overestimates actual disk usage — the definitive number comes from Nexus's *Admin → Repository → Repository Size* task.

---

## Reclaiming disk space

`DELETE /service/rest/v1/components/{id}` is a **soft delete** in Nexus: the component is unlinked immediately, but the underlying blobs stay on disk until compaction runs.

After large deletions, run the Nexus administrative task:

> **Admin → System → Tasks → Create Task → "Admin - Compact blob store"**

Until that task completes, disk usage will not decrease.

---

## Commands reference

```
nexus3-cli configure
nexus3-cli image ls
nexus3-cli image tags    --name <image>
nexus3-cli image info    --name <image> --tag <tag>
nexus3-cli image delete  --name <image> --tag <tag>
nexus3-cli image delete  --name <image> --keep <N> [--keep-within <duration>] [--yes]
nexus3-cli image size    --name <image>
nexus3-cli --version
```

---

## Requirements on the Nexus side

Because the CLI uses the Nexus REST API under Basic auth:

1. The user account needs these privileges on the target repository:
   - `nx-repository-view-docker-<repo>-read` (for `ls`, `tags`, `info`, `size`)
   - `nx-repository-view-docker-<repo>-delete` (for `delete`)
2. **No Docker connector, no Bearer Token Realm, no `Allow redeploy`** setting is required — those are only relevant to `docker pull/push`, which this tool does not perform.
3. Optionally verify the REST API is reachable:
   ```bash
   curl -u admin:PASS http://nexus:8081/service/rest/v1/status
   ```

---

## Development

```bash
make fmt      # gofmt
make lint     # go vet (+ golangci-lint if installed)
make test     # go test ./...
make run      # build and run ./dist/nexus3-cli
make clean    # remove dist/
```

Project layout:

```
main.go
internal/
  cli/         urfave/cli v2 command wiring
  config/      ~/.nexus-cli load/save
  registry/    Docker Registry v2 HTTP client
```

---

## License

MIT (or whatever the repository owner chooses — update this section before releasing).