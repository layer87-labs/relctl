// export_test.go exposes internal fields of CalVerScheme for white-box testing.
// This file is only compiled when running tests (the "_test.go" suffix).
package versioning

import "time"

// SetCalVerClock injects a custom clock into s. Use in tests to freeze time.
func SetCalVerClock(s *CalVerScheme, fn func() time.Time) {
	s.now = fn
}

// SetCalVerTagsFunc injects a custom tag-listing function into s.
// Use in tests to avoid touching the filesystem.
func SetCalVerTagsFunc(s *CalVerScheme, fn func(path string) ([]string, error)) {
	s.tagsForRepo = fn
}
