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
	Assert(t, pathMatch("/a", "/a"), true)
	Assert(t, pathMatch("/a/b", "/a/b"), true)
	Assert(t, pathMatch("/a/b/c", "/a/b/c"), true)

	// Flexible matching
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

	// Test normalization
	Assert(t, pathMatch("///", "/"), true)
	Assert(t, pathMatch("/a///b", "/a/b"), true)
	Assert(t, pathMatch("/a/////b/c", "/a/b/c"), true)
	Assert(t, pathMatch("/a/b/c///", "/a/b/c"), true)
	Assert(t, pathMatch("/a//b//c", "/a/b/c"), true)
}

func TestPathMatchComplexPaths(t *testing.T) {
	// Matching with placeholders
	Assert(t, pathMatch("/users/123", "/users/[id]"), true)
	Assert(t, pathMatch("/products/456/details", "/products/[productId]/details"), true)
	Assert(t, pathMatch("/shop/categories/electronics/items/789", "/shop/categories/[category]/items/[itemId]"), true)
	Assert(t, pathMatch("/posts/2023/november", "/posts/[year]/[month]"), true)
	Assert(t, pathMatch("/companies/abc/employees/john-doe", "/companies/[company]/employees/[employee]"), true)

	// Flexible placeholders with extra segments matching
	Assert(t, pathMatch("/users/123/settings", "/users/[id]"), true)
	Assert(t, pathMatch("/shop/categories/electronics/items/789/reviews", "/shop/categories/[category]/items/[itemId]"), true)
	Assert(t, pathMatch("/posts/2023/november/drafts", "/posts/[year]/[month]"), true)
	Assert(t, pathMatch("/companies/abc/employees/john-doe/profile", "/companies/[company]/employees/[employee]"), true)

	// Mismatched paths
	Assert(t, pathMatch("/users", "/users/[id]"), false)
	Assert(t, pathMatch("/products/456", "/products/[productId]/details"), false)
	Assert(t, pathMatch("/shop/categories/electronics/items", "/shop/categories/[category]/items/[itemId]"), false)
	Assert(t, pathMatch("/posts/2023", "/posts/[year]/[month]"), false)
	Assert(t, pathMatch("/companies/abc/employees", "/companies/[company]/employees/[employee]"), false)
	Assert(t, pathMatch("/products/456/reviews/789", "/products/[productId]/details"), false)
}
