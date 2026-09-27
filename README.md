# SourceAnt agent

The process that stays up on a developer's machine. It supervises the SourceAnt indexer, keeps the local code graph current, and serves it to the CLI and the local UI.

It never parses code. The grammars and the graph shape live in the Python core, so that one index serves every client.

## What it does

Starts the core on a free port and waits for it to answer. Restarts it when it dies, backing off as failures repeat. Serves its own HTTP surface on `127.0.0.1:8930`, where `/health` reports whether the core is up and how many times it has been started, and `/api/repositories` and `/api/graph` read the index.

It also serves the local view at `/`: Overview, Reviews, Skills, Knowledge, Graphs, Repositories, Settings. The assets are embedded in the binary, so the view works with no network and cannot drift from the agent serving it.

Ctrl or Cmd K finds a skill, a decision, a file or a repository by name from any screen. The smaller kinds are held and narrowed here; the code is narrowed by the core, because a graph is thousands of nodes.

The view is Vue, built by Vite from `ui/`: Tailwind for the design tokens, lucide for icons, force-graph in 2D and 3d-force-graph for Tree, Radial, Layered and Force.

A checkout is reviewed here, by the same reviewer a pull request goes through. It reads the working tree against the branch it would merge into, so a change is read before anybody else sees it. The core does the reading; the agent asks for it and keeps the answer.

`/mcp` is proxied through to the core, so a coding agent on this machine reaches the same index and can ask for the same review.

Loopback is the default because the agent reads a working tree. The machine it runs on is the only audience it has.

## Running it

Building needs Go and npm: `make build` builds the view, then the binary that embeds it. `make build-go` skips the view when nothing about it changed.

To work on the view, leave an agent running and start `make ui-dev` beside it. The dev server hands `/api` and `/health` to the agent on `127.0.0.1:8930`, or wherever `SOURCEANT_AGENT_URL` points, and reloads as the sources change. A change only reaches the binary through `make build`, since what it serves is the view embedded when it was built.

```bash
make build
./sourceant-agent
```

It starts whatever `sourceant setup` put on this machine, reading `~/.sourceant/config.json`: a container, or the core as a program. A container binds every interface inside and is published to loopback outside, because one binding the container's own loopback could be reached by nothing.

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
