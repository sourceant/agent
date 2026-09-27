# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0-beta.4] - 2026-09-27

### Added

- Ctrl or Cmd K finds a skill, a decision, a file or a repository by name from
  any screen, asking the core for the code because a graph is too large to hold
  on a screen
- The Overview leads with the one thing worth doing, and carries the last few
  reviews and the files that change most and are depended on most
- Folders sit under the owner their name carries, say when each was last read,
  and show a read as it happens
- Adding a folder searches for one by name rather than walking the tree
- Which uses a skill has is answered from the list, for every use and every
  skill, including the folders a coding agent syncs and nothing here may edit
- Reading a skill and changing one are two screens
- The model is chosen from what the core can reach, and saving a key says
  whether the provider accepts the pair

### Fixed

- A query for a drawing reaches the core, which narrows it, rather than being
  applied to whatever had already been sent
- Opening a review no longer narrows the listing to that review's repository
- Choosing all repositories survives a screen that can only show one
- The view is built against the design package it is locked to

## [1.0.0-beta.3]

### Added

- Local stack shutdown waits for core cleanup and only stops this agent's container.
- Graph loading and failure states explain the wait and allow retrying.
- Model setup offers an optional skip, including when settings cannot load.

### Fixed

- Large graphs display their group controls in batches.
- Adding a folder indexes only that repository.

### Compatibility

- Core `1.0.0-beta.2` is the compatibility baseline for this release.
- Newer core beta releases require compatibility checks before being added to the supported set.

## [1.0.0-beta.2] - 2026-08-30

First release, versioned alongside the core it supervises.

### Added

- Supervises the core: starts whatever was installed on a free port, waits for
  it to answer, and restarts it when it dies, backing off as failures repeat
- Serves the local view from the binary itself, so it works with no network and
  cannot drift from the agent serving it: Overview, Graph, Knowledge, Reviews,
  Skills, Repositories and Settings
- Asks for a review of a checkout and keeps the answer, so a review has an
  address that still opens an hour later
- Proxies `/mcp` to the core, so a coding agent on this machine reaches the same
  index through the agent it already talks to
- Reads and writes knowledge, skills, settings and repositories against the
  index, and browses the filesystem for the folder a person is adding
