# Versions

The agent and core are versioned independently.

For agent `1.0.0-beta.3`, use core `1.0.0-beta.2`:

```bash
sourceant setup --agent-version 1.0.0-beta.3 --core-version 1.0.0-beta.2
```

The agent checks whether the core is running, but does not enforce a version range.
