# SourceAnt agent

The process that stays up on a developer's machine. It supervises the SourceAnt indexer, keeps the local code graph current, and serves it to the CLI and the local UI.

It never parses code. The grammars and the graph shape live in the Python core, so that one index serves every client.

## What it does

Starts the core on a free port and waits for it to answer. Restarts it when it dies, backing off as failures repeat. Serves its own HTTP surface on `127.0.0.1:8930`, where `/health` reports whether the core is up and how many times it has been started, and `/api/repositories` and `/api/graph` read the index.

It also serves the local view at `/`: Overview, Knowledge graph, Knowledge, Repositories, Settings. The assets are embedded in the binary, so the view works with no network and cannot drift from the agent serving it.

The view is Vue, built by Vite from `ui/`, and it takes the dashboard's design system rather than imitating it: the same tokens, the same Card, Button, Badge, Avatar and Logo, lucide for icons, force-graph in 2D and 3d-force-graph for Tree, Radial, Layered and Force. Copying a design by hand is what let the two drift apart the first time.

There are no reviews here. A review reads a pull request, which is a thing the hosted service does.

What opens is the repository's shape: folders and files. Symbols, imports and the test suite are there to be asked for, because a repository's every function is a texture rather than a picture. The folder nodes are this view's arrangement, read out of the paths the index already carries; the index stores no folders of its own.

Loopback is the default because the agent reads a working tree. The machine it runs on is the only audience it has.

## Running it

Building needs Go and npm: `make build` builds the view, then the binary that embeds it. `make build-go` skips the view when nothing about it changed, and `make ui-dev` runs it against an agent already running.

```bash
make build
./sourceant-agent
```

It starts whatever `sourceant install` put on this machine, reading `~/.sourceant/config.json`: a container, or the core as a program. A container binds every interface inside and is published to loopback outside, because one binding the container's own loopback could be reached by nothing.

With nothing installed it looks for `sourceant` on `PATH`, which is what somebody working on the core itself already has. See [sourceant/cli](https://github.com/sourceant/cli) to install one, and [sourceant/sourceant](https://github.com/sourceant/sourceant) for the core.

| Variable | Default | Meaning |
|---|---|---|
| `SOURCEANT_AGENT_LISTEN` | `127.0.0.1:8930` | Where the agent answers |
| `SOURCEANT_CORE` | what was installed | A core to supervise instead, overriding the install |
| `SOURCEANT_CORE_PORT` | chosen at start | The port to start the core on |

## Building

```bash
make qa      # fmt-check, vet, lint, test
make build
```

## Licence

MIT.
