package sourceaddrs

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// normalizeSubpath interprets the given string as a package "sub-path",
// returning a normalized form of the path or an error if the string does
// not use correct syntax.
func normalizeSubpath(given string) (string, error) {
	if given == "" {
		// The empty string is how we represent the absense of a subpath,
		// which represents the root directory of a package.
		return "", nil
	}

	clean := path.Clean(given)

	// Our definition of "sub-path" aligns with the definition used by Go's
	// virtual filesystem abstraction, since our "module package" idea
	// is also essentially just a virtual filesystem.
	// This definition prohibits "." and ".." segments and therefore prevents
	// upward path traversal.
	// Go's path wrangling uses "." to represent "root directory", but
	// we represent that by omitting the subpath entirely, so we forbid that
	// too even though Go would consider it valid.
	if clean == "." || !fs.ValidPath(clean) {
		return "", fmt.Errorf("must be slash-separated relative path without any .. or . segments")
	}

	return clean, nil
}

// subPathAsLocalSource interprets the given subpath (which should be a value
// previously returned from [normalizeSubpath]) as a local source address
// relative to the root of the package that the sub-path was presented against.
func subPathAsLocalSource(p string) LocalSource {
	// Local source addresses are _mostly_ a superset of what we allow in
	// sub-paths, except that downward traversals must always start with
	// "./" to disambiguate from other address types.
	return LocalSource{relPath: "./" + p}
}

// splitSubPath takes a source address that would be accepted either as a
// remote source address or a registry source address and returns a tuple of
// its package address and its sub-path portion.
//
// For example:
//   dom.com/path/?q=p               => "dom.com/path/?q=p", ""
//   proto://dom.com/path//*?q=p     => "proto://dom.com/path?q=p", "*"
//   proto://dom.com/path//path2?q=p => "proto://dom.com/path?q=p", "path2"
//
// This function DOES NOT validate or normalize the sub-path. Pass the second
// return value to [normalizeSubpath] to check if it is valid and to obtain
// its normalized form.
func splitSubPath(src string) (string, string) {
	// This is careful to handle the query string portion of a remote source
	// address. That's not actually necessary for a module registry address
	// because those don't have query strings anyway, but it doesn't _hurt_
	// to check for a query string in that case and allows us to reuse this
	// function for both cases.

	// URL might contains another url in query parameters
	stop := len(src)
	if idx := strings.Index(src, "?"); idx > -1 {
		stop = idx
	}

	// Calculate an offset to avoid accidentally marking the scheme
	// as the dir.
	var offset int
	if idx := strings.Index(src[:stop], "://"); idx > -1 {
		offset = idx + 3
	}

	// First see if we even have an explicit subdir
	idx := strings.Index(src[offset:stop], "//")
	if idx == -1 {
		return src, ""
	}

	idx += offset
	subdir := src[idx+2:]
	src = src[:idx]

	// Next, check if we have query parameters and push them onto the
	// URL.
	if idx = strings.Index(subdir, "?"); idx > -1 {
		query := subdir[idx:]
		subdir = subdir[:idx]
		src += query
	}

	return src, subdir
}
