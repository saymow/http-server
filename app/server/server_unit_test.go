package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathMatch(t *testing.T) {
	// Matching
	assert.True(t, pathMatch("/", "/"))
	assert.True(t, pathMatch("/a", "/a"))
	assert.True(t, pathMatch("/a/b", "/a/b"))
	assert.True(t, pathMatch("/a/b/c", "/a/b/c"))

	// Test wildcard
	assert.True(t, pathMatch("/a", "*"))
	assert.True(t, pathMatch("/a/b", "*"))
	assert.True(t, pathMatch("/a/b/c", "*"))
	assert.True(t, pathMatch("/a", "/*"))
	assert.True(t, pathMatch("/a/b", "/*"))
	assert.True(t, pathMatch("/a/b/c", "/*"))
	assert.True(t, pathMatch("/a/b/c", "/a/*"))
	assert.True(t, pathMatch("/a/b/c", "/a/b/*"))

	// Test normalization
	assert.True(t, pathMatch("///", "/"))
	assert.True(t, pathMatch("/a///b", "/a/b"))
	assert.True(t, pathMatch("/a/////b/c", "/a/b/c"))
	assert.True(t, pathMatch("/a/b/c///", "/a/b/c"))
	assert.True(t, pathMatch("/a//b//c", "/a/b/c"))

	// Non-matching paths
	assert.False(t, pathMatch("/", "/a"))
	assert.False(t, pathMatch("/", "/a/b"))
	assert.False(t, pathMatch("/", "/a/b/c"))
	assert.False(t, pathMatch("/a", "/b"))
	assert.False(t, pathMatch("/a/b", "/b"))
	assert.False(t, pathMatch("/a/b/c", "/b/c"))
	assert.False(t, pathMatch("/x/y/z", "/a/b/c"))
	assert.False(t, pathMatch("/a/b/c/d", "/a/b/c"))

	// Partial path matches
	assert.False(t, pathMatch("/a/b/c", "/a/b/c/d"))
	assert.False(t, pathMatch("/a/b", "/a/b/c"))
	assert.False(t, pathMatch("/a", "/a/b/c"))
	assert.False(t, pathMatch("/a/b/c", "/a/b/x"))
	assert.False(t, pathMatch("/a", "/"))
	assert.False(t, pathMatch("/a/b", "/"))
	assert.False(t, pathMatch("/a/b/c", "/"))
}

func TestPathMatchComplexPaths(t *testing.T) {
	// Matching with placeholders
	assert.True(t, pathMatch("/users/123", "/users/[id]"))
	assert.True(t, pathMatch("/products/456/details", "/products/[productId]/details"))
	assert.True(t, pathMatch("/shop/categories/electronics/items/789", "/shop/categories/[category]/items/[itemId]"))
	assert.True(t, pathMatch("/posts/2023/november", "/posts/[year]/[month]"))
	assert.True(t, pathMatch("/companies/abc/employees/john-doe", "/companies/[company]/employees/[employee]"))

	// Mismatched paths
	assert.False(t, pathMatch("/users", "/users/[id]"))
	assert.False(t, pathMatch("/products/456", "/products/[productId]/details"))
	assert.False(t, pathMatch("/shop/categories/electronics/items", "/shop/categories/[category]/items/[itemId]"))
	assert.False(t, pathMatch("/posts/2023", "/posts/[year]/[month]"))
	assert.False(t, pathMatch("/companies/abc/employees", "/companies/[company]/employees/[employee]"))
	assert.False(t, pathMatch("/products/456/reviews/789", "/products/[productId]/details"))

	// Mismatched paths (different structures)
	assert.False(t, pathMatch("/users/123/settings", "/users/[id]"))
	assert.False(t, pathMatch("/shop/categories/electronics/items/789/reviews", "/shop/categories/[category]/items/[itemId]"))
	assert.False(t, pathMatch("/posts/2023/november/drafts", "/posts/[year]/[month]"))
	assert.False(t, pathMatch("/companies/abc/employees/john-doe/profile", "/companies/[company]/employees/[employee]"))
}

func TestGetRouteParams(t *testing.T) {
	// Basic matching with one parameter
	params := getRouteParams("/users/5", "/users/[id]")
	assert.Equal(t, len(params), 1)
	assert.Equal(t, params["id"], "5")

	// Matching with multiple parameters
	params = getRouteParams("/products/123/details", "/products/[productId]/[section]")
	assert.Equal(t, len(params), 2)
	assert.Equal(t, params["productId"], "123")
	assert.Equal(t, params["section"], "details")

	// Matching with nested parameters
	params = getRouteParams("/companies/abc/employees/42", "/companies/[company]/employees/[employeeId]")
	assert.Equal(t, len(params), 2)
	assert.Equal(t, params["company"], "abc")
	assert.Equal(t, params["employeeId"], "42")

	// Matching with parameters and additional static segments
	params = getRouteParams("/users/5/settings", "/users/[id]/settings")
	assert.Equal(t, len(params), 1)
	assert.Equal(t, params["id"], "5")

	// Placeholder at the end of the path
	params = getRouteParams("/posts/2023", "/posts/[year]")
	assert.Equal(t, len(params), 1)
	assert.Equal(t, params["year"], "2023")

	// Multiple consecutive placeholders
	params = getRouteParams("/data/123/456", "/data/[param1]/[param2]")
	assert.Equal(t, len(params), 2)
	assert.Equal(t, params["param1"], "123")
	assert.Equal(t, params["param2"], "456")

	// Multiple consecutive placeholders 2
	params = getRouteParams("/data/123/456/test/789", "/data/[param1]/[param2]/test/[param3]")
	assert.Equal(t, len(params), 3)
	assert.Equal(t, params["param1"], "123")
	assert.Equal(t, params["param2"], "456")
	assert.Equal(t, params["param3"], "789")

	// Non-matching path (different structure)
	params = getRouteParams("/products/123", "/users/[id]")
	assert.Equal(t, len(params), 0) // No params since paths are different

	// Non-matching path (different structure)
	params = getRouteParams("/users/5/details", "/users/[id]")
	assert.Equal(t, len(params), 0) // No params since paths are different

	// Edge case with no parameters
	params = getRouteParams("/static/path", "/static/path")
	assert.Equal(t, len(params), 0) // No params expected as there are no placeholders
}
