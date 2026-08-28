# SourceAnt agent

The process that stays up on a developer's machine. It supervises the SourceAnt indexer, keeps the local code graph current, and serves it to the CLI and the local UI.

It never parses code. The grammars and the graph shape live in the Python core, so that one index serves every client.

## What it does

Starts the core on a free port and waits for it to answer. Restarts it when it dies, backing off as failures repeat. Serves its own HTTP surface on `127.0.0.1:8930`, where `/health` reports whether the core is up and how many times it has been started, and `/api/repositories` and `/api/graph` read the index.

It also serves the graph view at `/`. The assets are embedded in the binary, so the view works with no network and cannot drift from the agent serving it.

What opens is the repository's shape: folders and files. Symbols, imports and the test suite are there to be asked for, because a repository's every function is a texture rather than a picture. The folder nodes are this view's arrangement, read out of the paths the index already carries; the index stores no folders of its own.

Loopback is the default because the agent reads a working tree. The machine it runs on is the only audience it has.

## Running it

The core has to be on `PATH` as `sourceant`. See [sourceant/sourceant](https://github.com/sourceant/sourceant).

```bash
make build
./sourceant-agent
```

| Variable | Default | Meaning |
|---|---|---|
| `SOURCEANT_AGENT_LISTEN` | `127.0.0.1:8930` | Where the agent answers |
| `SOURCEANT_CORE` | `sourceant` | The core executable to supervise |
| `SOURCEANT_CORE_PORT` | chosen at start | The port to start the core on |

## Building

```bash
make qa      # fmt-check, vet, lint, test
make build
```

## Licence

MIT.
