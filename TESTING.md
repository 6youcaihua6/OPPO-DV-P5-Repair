# Validation for 0.1BETA

29 top-level automated tests passed with real-tool integration enabled. One opt-in real P5 source preflight was skipped. Test fixtures use synthetic media rather than a full Dolby Vision feature film.

Coverage includes parameter-header parsing, safe preflight, Unicode paths, non-overwriting output publication, timestamps, directory recursion and deduplication, single-instance locking, task isolation, cancellation, live log language rendering, all 12 report languages and sponsor layout geometry.

Real FFmpeg/ffprobe and MKVToolNix checks cover synthetic HEVC inspection/export, corrupt input, unchanged source hashes, temporary-file cleanup, and a remux round trip with 12 audio tracks and a PNG attachment.

## Reproduce

```powershell
go test -v ./...
```

For real-tool tests, set `P5_INTEGRATION=1`, `P5_FFMPEG`, `P5_FFPROBE`, and `P5_MKV_DIR` to your installed tool paths before running the suite.

## Not verified

- Full interactive desktop acceptance, cross-monitor DPI operation.
- Full-film P5 repair and OPPO/display playback.
- `go vet` reports an existing unsafe-pointer warning in the Win32 `WM_DPICHANGED` handler. The Windows-provided rectangle is read during message handling; the static analysis run is not described as clean.

Passing tests does not prove the absence of defects.
