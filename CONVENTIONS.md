# Project Conventions

Every project in `projects/` follows these rules so the repo stays consistent across 365 days and a dozen languages.

## Folder layout

```
projects/NNN-kebab-title/
├── README.md        # from templates/PROJECT_README.md
├── <build files>    # package.json, Cargo.toml, go.mod, pyproject.toml, CMakeLists.txt, .csproj, ...
├── src/             # source (or the ecosystem's idiomatic equivalent)
│   ├── core/        # pure game logic: no rendering, audio, or window code
│   └── ...          # platform/presentation layer
└── tests/           # automated tests (or the ecosystem's idiomatic location)
```

- `NNN` is the zero-padded day number from [PROJECTS.md](PROJECTS.md).
- Use the stack listed for that day exactly.

## Architecture

1. **Separate the simulation from the presentation.** Game rules, state, and algorithms live in a core module with no dependency on the graphics/audio framework. The renderer reads state and draws it. This keeps the logic testable and portable.
2. **Make time explicit.** Update functions take `dt` (or a fixed step) as a parameter. Don't read the wall clock inside game logic.
3. **Make randomness injectable.** Pass seeded RNGs in so behavior is reproducible in tests.
4. **No dead code, no commented-out blocks, no TODO stubs** posing as features.

## Testing

- Core logic has unit tests, and they must run with the ecosystem's standard command (`npm test`, `pytest`, `cargo test`, `go test ./...`, `dotnet test`, `ctest`, `busted`, `gradle test`, ...).
- Prefer behavior-level tests: "ball bounces off paddle," "A* finds the shortest path," "replay reproduces the same final state."
- If a toolchain truly can't run headlessly (for example, Godot), still write the tests and say so in the README.

## README requirements

Every project README covers:
1. **What it is**, with a one-line pitch and a screenshot or ASCII diagram where useful
2. **Gaming technique**: how the feature works and why games use it
3. **Software engineering concepts**: patterns, architecture, and trade-offs
4. **Architecture**: a Mermaid or ASCII sketch
5. **Build, run, and test** commands
6. **Possible extensions**

## Scope by level

| Level | Expectation |
|-------|-------------|
| Starter | Small, polished, single focus. A few files, well tested. |
| Intermediate | Multi-module project with a clear architecture and solid test coverage. |
| Advanced | Serious implementation with real depth, benchmarks or stress tests where relevant. |

## Hygiene

- No secrets, API keys, or real payment credentials. Mock external services.
- No large binaries. Prefer procedurally drawn shapes or tiny generated assets, and only use assets with a public-domain or CC0 license.
- No commercial game ROMs or copyrighted assets.
- Networking and security projects run against localhost only.
- Commit message: `Add project NNN: Title`
