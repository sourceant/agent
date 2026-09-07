# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0-beta.3]

### Added

- Local stack shutdown waits for core cleanup and only stops this agent's container.
- Graph loading and failure states explain the wait and allow retrying.
- Model setup offers an optional skip, including when settings cannot load.

### Fixed

- Large graphs display their group controls in batches.
- Adding a folder indexes only that repository.

### Compatibility

- Core compatibility: `1.0.0-beta.2`.

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
