# URL Building Patterns Guide

This document describes common patterns for building URLs safely in Go and which patterns to avoid.

## ✅ Safe Patterns (Use These)

### 1. Using `utils.JoinURL()` (Recommended)
```go
import "github.com/forkbombeu/credimi/pkg/utils"

url := utils.JoinURL(baseURL, "api", "v1", "users")
// Result: "https://example.com/api/v1/users"
```

**Use when:**
- Building URL paths from a base URL and path segments
- Multiple path segments need to be joined
- You want automatic handling of trailing slashes

### 2. Using `url.JoinPath()` (Standard Library)
```go
import "net/url"

path, _ := url.JoinPath("/api", "v1", "users")
// Result: "/api/v1/users"
```

**Use when:**
- Joining URL path segments (not full URLs)
- You only have path components, not a full base URL

### 3. Using `url.Parse()` + `url.URL` struct
```go
import "net/url"

u, _ := url.Parse("https://example.com")
u.Path, _ = url.JoinPath(u.Path, "api", "v1")
u.RawQuery = "key=value"
finalURL := u.String()
```

**Use when:**
- You need to modify query parameters
- You need fine-grained control over URL components
- Building URLs with query strings

## ❌ Unsafe Patterns (Avoid These)

### 1. String Concatenation with `+`
```go
// ❌ BAD
url := baseURL + "/api/v1/users"
url := "https://" + domain + "/path"

// ✅ GOOD
url := utils.JoinURL(baseURL, "api", "v1", "users")
```

**Problems:**
- Doesn't handle trailing slashes correctly
- Can create double slashes
- Doesn't encode special characters
- Error-prone with edge cases

### 2. `fmt.Sprintf()` for Path Building
```go
// ❌ BAD
url := fmt.Sprintf("%s/api/v1/%s", baseURL, userID)

// ✅ GOOD
url := utils.JoinURL(baseURL, "api", "v1", userID)
```

**Problems:**
- Same issues as string concatenation
- Harder to read with multiple segments
- No URL encoding

### 3. `path.Join()` or `filepath.Join()` for URLs
```go
// ❌ BAD
import "path"
url := path.Join(baseURL, "api", "v1")

// ✅ GOOD
import "net/url"
url, _ := url.JoinPath(baseURL, "api", "v1")
// OR
url := utils.JoinURL(baseURL, "api", "v1")
```

**Problems:**
- `path.Join()` is for file paths, not URLs
- Doesn't handle URL schemes correctly
- Can break on Windows (uses backslashes)

### 4. `strings.Join()` with "/"
```go
// ❌ BAD
parts := []string{"api", "v1", "users"}
url := baseURL + "/" + strings.Join(parts, "/")

// ✅ GOOD
url := utils.JoinURL(baseURL, parts...)
```

**Problems:**
- Manual slash handling
- Doesn't handle base URL trailing slashes
- No URL encoding

### 5. Manual Query Parameter Building
```go
// ❌ BAD
url := baseURL + "?key1=" + value1 + "&key2=" + value2

// ✅ GOOD
u, _ := url.Parse(baseURL)
q := u.Query()
q.Set("key1", value1)
q.Set("key2", value2)
u.RawQuery = q.Encode()
url := u.String()
```

**Problems:**
- No URL encoding
- Special characters break URLs
- Error-prone

## Common Use Cases

### Building API Endpoints
```go
// ✅ GOOD
endpoint := utils.JoinURL(apiBaseURL, "api", "v1", "users", userID)
```

### Adding Query Parameters
```go
// ✅ GOOD
u, _ := url.Parse(baseURL)
q := u.Query()
q.Set("page", "1")
q.Set("limit", "10")
u.RawQuery = q.Encode()
finalURL := u.String()
```

### Building URLs from Config
```go
// ✅ GOOD
appURL := config["app_url"].(string)
testURL := utils.JoinURL(appURL, "my", "tests", "runs", runID)
```

### Normalizing URLs (Adding Scheme)
```go
// For adding scheme, string concatenation is acceptable if you validate
cleanURL := strings.TrimSpace(baseURL)
if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
    cleanURL = "https://" + cleanURL
}
// Then use utils.JoinURL for paths
```

## When to Use Each Pattern

| Pattern | Use Case |
|---------|----------|
| `utils.JoinURL()` | Building URLs from base + path segments (most common) |
| `url.JoinPath()` | Joining path segments only (no base URL) |
| `url.Parse()` + `url.URL` | Complex URLs with query params, fragments, etc. |
| String concat | Only for simple cases like adding scheme prefix (with validation) |

## Migration Checklist

When refactoring existing code:

1. ✅ Find all `+ "/"` patterns
2. ✅ Find all `fmt.Sprintf` with URL patterns
3. ✅ Find all `path.Join` used with URLs
4. ✅ Find all `strings.Join` with "/" for URLs
5. ✅ Replace with `utils.JoinURL()` or `url.JoinPath()`
6. ✅ Test edge cases (trailing slashes, special characters)
7. ✅ Run the search script: `./scripts/find_manual_urls.sh`

## Search Script

Use `./scripts/find_manual_urls.sh` to find potential manual URL building patterns in the codebase.

