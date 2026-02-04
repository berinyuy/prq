# Release

Releases are handled by GoReleaser.

## Tag a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Artifacts

- Linux, macOS, Windows binaries
- Checksums
- Homebrew tap formula
- Scoop manifest

Update `.goreleaser.yaml` with your org names for Homebrew/Scoop.
