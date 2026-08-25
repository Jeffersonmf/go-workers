// Package util holds small, dependency-light helpers shared across the
// module: a package-wide structured logger, environment/file-based
// configuration, and UUID generation.
package util

import "go.uber.org/zap"

// Sugar is the package-wide logger. It is set by a package-level
// variable initializer, not an init() func, and that choice is load
// bearing: the Go spec guarantees every package-level variable
// initializer runs before any init() function in the same package,
// regardless of which file declares them. config_manager.go's init()
// logs through Sugar too, and Go otherwise only guarantees init()
// order within a single file — across files it follows the order the
// build presents them in, conventionally lexical filename order
// ("config_manager.go" before "logger.go"). Setting Sugar inside its
// own init() worked by accident of that ordering; a rename, a new
// file starting with a letter before "l", or a different toolchain
// would have reintroduced a nil-pointer panic during startup.
var Sugar = newSugaredLogger()

func newSugaredLogger() *zap.SugaredLogger {
	logger, err := zap.NewProduction()
	if err != nil {
		fallback := zap.NewNop().Sugar()
		fallback.Errorf("falling back to a no-op logger: %v", err)
		return fallback
	}
	return logger.Sugar()
}

// Sync flushes any buffered log entries. Callers should defer it in
// main after the logger has been used, as recommended by zap.
func Sync() {
	if err := Sugar.Sync(); err != nil {
		// Sync commonly fails on stderr/stdout with "invalid argument"
		// on some platforms; it is not actionable, so it is reported
		// but not treated as fatal.
		Sugar.Debugf("logger sync: %v", err)
	}
}
