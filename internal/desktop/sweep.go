//go:build desktop

package desktop

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// staleUserDataAge is how old an orphaned webview user-data folder must be
// before the boot sweep will remove it. Fresh folders are skipped so a
// concurrently starting shell — which creates its folder before its webview
// opens handles inside — is never raced.
const staleUserDataAge = time.Hour

// sweepStaleUserData removes webview user-data folders left behind by shells
// that crashed or were killed before their best-effort exit cleanup. It is
// best-effort and deliberately conservative:
//
//   - keep (this process's own folder) is never touched;
//   - only directories matching the shell's own webview-* naming are
//     considered, and nothing outside base is reached;
//   - a folder must be older than staleUserDataAge;
//   - the folder is renamed to a staging name first: while any live process
//     still holds a handle inside (typically a webview child of another
//     running shell), Windows refuses the rename and the folder is skipped —
//     unsure means skip;
//   - failures are logged and never fatal: a failed sweep must not stop the
//     shell from starting.
//
// It returns the folder names it removed, for the log line and the tests.
func sweepStaleUserData(base, keep string, now time.Time, log *slog.Logger) []string {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	entries, e := filepath.Glob(filepath.Join(base, "webview-*"))
	if e != nil {
		log.Debug("desktop: userdata sweep glob failed", "error", e)
		return nil
	}
	var swept []string
	keepClean := filepath.Clean(keep)
	for _, dir := range entries {
		if filepath.Clean(dir) == keepClean {
			continue // never the current one
		}
		info, e := os.Stat(dir)
		if e != nil || !info.IsDir() {
			continue
		}
		if now.Sub(info.ModTime()) < staleUserDataAge {
			continue // possibly a shell that is still coming up
		}
		staging := filepath.Join(base, ".sweep-"+randomID())
		if e := os.Rename(dir, staging); e != nil {
			log.Debug("desktop: userdata sweep skipped a live or busy folder", "dir", filepath.Base(dir), "error", e)
			continue
		}
		if e := os.RemoveAll(staging); e != nil {
			log.Warn("desktop: userdata sweep could not finish removal", "dir", filepath.Base(dir), "error", e)
			if e2 := os.Rename(staging, dir); e2 != nil {
				log.Warn("desktop: userdata sweep left a folder in staging", "dir", filepath.Base(dir), "error", e2)
			}
			continue
		}
		swept = append(swept, filepath.Base(dir))
		log.Info("desktop: swept stale webview data folder", "dir", filepath.Base(dir))
	}
	return swept
}
