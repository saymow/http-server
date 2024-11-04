package server

import "testing"

func Assert[T comparable](t *testing.T, received, expected T) {
	t.Helper()

	if expected != received {
		t.Errorf("Expected %v but recevied %v.", expected, received)
	}
}

func TestPathMatch(t *testing.T) {
	// Matching
	Assert(t, pathMatch("/", "/"), true)
	Assert(t, pathMatch("/a", "/"), true)
	Assert(t, pathMatch("/a/b", "/"), true)
	Assert(t, pathMatch("/a/b/c", "/"), true)

	// Non-matching paths
	Assert(t, pathMatch("/", "/a"), false)
	Assert(t, pathMatch("/", "/a/b"), false)
	Assert(t, pathMatch("/", "/a/b/c"), false)
	Assert(t, pathMatch("/a", "/b"), false)
	Assert(t, pathMatch("/a/b", "/b"), false)
	Assert(t, pathMatch("/a/b/c", "/b/c"), false)
	Assert(t, pathMatch("/x/y/z", "/a/b/c"), false)
	Assert(t, pathMatch("/a/b/c/d", "/a/b/c"), true)

	// Partial path matches
	Assert(t, pathMatch("/a/b/c", "/a/b/c/d"), false)
	Assert(t, pathMatch("/a/b", "/a/b/c"), false)
	Assert(t, pathMatch("/a", "/a/b/c"), false)
	Assert(t, pathMatch("/a/b/c", "/a/b/x"), false)
}
