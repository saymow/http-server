package server

import (
	"testing"
)

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

	// Test wildcard
	Assert(t, pathMatch("/a", "*"), true)
	Assert(t, pathMatch("/a/b", "*"), true)
	Assert(t, pathMatch("/a/b/c", "*"), true)
	Assert(t, pathMatch("/a", "/*"), true)
	Assert(t, pathMatch("/a/b", "/*"), true)
	Assert(t, pathMatch("/a/b/c", "/*"), true)
	Assert(t, pathMatch("/a/b/c", "/a/*"), true)
	Assert(t, pathMatch("/a/b/c", "/a/b/*"), true)

	// Test normalization
	Assert(t, pathMatch("///", "/"), true)
	Assert(t, pathMatch("/a///b", "/a/b"), true)
	Assert(t, pathMatch("/a/////b/c", "/a/b/c"), true)
	Assert(t, pathMatch("/a/b/c///", "/a/b/c"), true)
	Assert(t, pathMatch("/a//b//c", "/a/b/c"), true)

	// Non-matching paths
	Assert(t, pathMatch("/", "/a"), false)
	Assert(t, pathMatch("/", "/a/b"), false)
	Assert(t, pathMatch("/", "/a/b/c"), false)
	Assert(t, pathMatch("/a", "/b"), false)
	Assert(t, pathMatch("/a/b", "/b"), false)
	Assert(t, pathMatch("/a/b/c", "/b/c"), false)
	Assert(t, pathMatch("/x/y/z", "/a/b/c"), false)
	Assert(t, pathMatch("/a/b/c/d", "/a/b/c"), false)

	// Partial path matches
	Assert(t, pathMatch("/a/b/c", "/a/b/c/d"), false)
	Assert(t, pathMatch("/a/b", "/a/b/c"), false)
	Assert(t, pathMatch("/a", "/a/b/c"), false)
	Assert(t, pathMatch("/a/b/c", "/a/b/x"), false)
	Assert(t, pathMatch("/a", "/"), false)
	Assert(t, pathMatch("/a/b", "/"), false)
	Assert(t, pathMatch("/a/b/c", "/"), false)
}

func TestPathMatchComplexPaths(t *testing.T) {
	// Matching with placeholders
	Assert(t, pathMatch("/users/123", "/users/[id]"), true)
	Assert(t, pathMatch("/products/456/details", "/products/[productId]/details"), true)
	Assert(t, pathMatch("/shop/categories/electronics/items/789", "/shop/categories/[category]/items/[itemId]"), true)
	Assert(t, pathMatch("/posts/2023/november", "/posts/[year]/[month]"), true)
	Assert(t, pathMatch("/companies/abc/employees/john-doe", "/companies/[company]/employees/[employee]"), true)

	// Mismatched paths
	Assert(t, pathMatch("/users", "/users/[id]"), false)
	Assert(t, pathMatch("/products/456", "/products/[productId]/details"), false)
	Assert(t, pathMatch("/shop/categories/electronics/items", "/shop/categories/[category]/items/[itemId]"), false)
	Assert(t, pathMatch("/posts/2023", "/posts/[year]/[month]"), false)
	Assert(t, pathMatch("/companies/abc/employees", "/companies/[company]/employees/[employee]"), false)
	Assert(t, pathMatch("/products/456/reviews/789", "/products/[productId]/details"), false)

	// Mismatched paths (different structures)
	Assert(t, pathMatch("/users/123/settings", "/users/[id]"), false)
	Assert(t, pathMatch("/shop/categories/electronics/items/789/reviews", "/shop/categories/[category]/items/[itemId]"), false)
	Assert(t, pathMatch("/posts/2023/november/drafts", "/posts/[year]/[month]"), false)
	Assert(t, pathMatch("/companies/abc/employees/john-doe/profile", "/companies/[company]/employees/[employee]"), false)
}

func TestGetRouteParams(t *testing.T) {
	// Basic matching with one parameter
	params := getRouteParams("/users/5", "/users/[id]")
	Assert(t, len(params), 1)
	Assert(t, params["id"], "5")

	// Matching with multiple parameters
	params = getRouteParams("/products/123/details", "/products/[productId]/[section]")
	Assert(t, len(params), 2)
	Assert(t, params["productId"], "123")
	Assert(t, params["section"], "details")

	// Matching with nested parameters
	params = getRouteParams("/companies/abc/employees/42", "/companies/[company]/employees/[employeeId]")
	Assert(t, len(params), 2)
	Assert(t, params["company"], "abc")
	Assert(t, params["employeeId"], "42")

	// Matching with parameters and additional static segments
	params = getRouteParams("/users/5/settings", "/users/[id]/settings")
	Assert(t, len(params), 1)
	Assert(t, params["id"], "5")

	// Placeholder at the end of the path
	params = getRouteParams("/posts/2023", "/posts/[year]")
	Assert(t, len(params), 1)
	Assert(t, params["year"], "2023")

	// Multiple consecutive placeholders
	params = getRouteParams("/data/123/456", "/data/[param1]/[param2]")
	Assert(t, len(params), 2)
	Assert(t, params["param1"], "123")
	Assert(t, params["param2"], "456")

	// Multiple consecutive placeholders 2
	params = getRouteParams("/data/123/456/test/789", "/data/[param1]/[param2]/test/[param3]")
	Assert(t, len(params), 3)
	Assert(t, params["param1"], "123")
	Assert(t, params["param2"], "456")
	Assert(t, params["param3"], "789")

	// Non-matching path (different structure)
	params = getRouteParams("/products/123", "/users/[id]")
	Assert(t, len(params), 0) // No params since paths are different

	// Non-matching path (different structure)
	params = getRouteParams("/users/5/details", "/users/[id]")
	Assert(t, len(params), 0) // No params since paths are different

	// Edge case with no parameters
	params = getRouteParams("/static/path", "/static/path")
	Assert(t, len(params), 0) // No params expected as there are no placeholders
}
