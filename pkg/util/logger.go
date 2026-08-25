// Package util holds small, dependency-light helpers shared across the
// module: a package-wide structured logger, environment/file-based
// configuration, and UUID generation.
package util

import (
	"log/slog"
	"os"
)

// Logger is the package-wide structured logger, JSON to stdout at Info
// level by default.
//
// This used to be a *zap.SugaredLogger built by zap.NewProduction(),
// which can fail (returning an error) and needed a fallback path
// (see docs/trade-offs.md #2 for the nil-pointer bug that path was
// guarding against). slog.New never fails: there is no error to
// handle and no fallback logger to construct, so the entire bug class
// that motivated moving initialization out of an init() no longer has
// a way to occur. The rename from Sugar to Logger follows the type:
// zap's "Sugared" logger took printf-style or key-value args
// interchangeably; slog.Logger is key-value only, which every call
// site in this module already used.
var Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))
