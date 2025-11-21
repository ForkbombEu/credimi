# URL Building Refactoring Summary

## Overview
This document summarizes the refactoring effort to replace manual URL building with safer utilities.

## Files Updated (10 total)

### Workflow Files (6 files)
1. **pkg/workflowengine/workflows/eudiw.go**
   - Replaced `fmt.Sprintf` for TemporalUI URLs
   - Replaced string concatenation for `/tests/wallet/eudiw`
   - Replaced `fmt.Sprintf` for `/api/compliance/send-eudiw-log-update`

2. **pkg/workflowengine/workflows/mobile.go**
   - Replaced `fmt.Sprintf` for testRunURL
   - Replaced `fmt.Sprintf` for mobile server URL path

3. **pkg/workflowengine/workflows/conformance_check.go**
   - Replaced `fmt.Sprintf` for base URL with suite path

4. **pkg/workflowengine/workflows/wallet.go**
   - Replaced `fmt.Sprintf` for TemporalUI URL

5. **pkg/workflowengine/workflows/vlei.go**
   - Replaced `fmt.Sprintf` for TemporalUI URLs (2 occurrences)
   - Replaced `fmt.Sprintf` for oobi endpoint URL

6. **pkg/workflowengine/pipeline/mobile_automation_hooks.go**
   - Replaced `fmt.Sprintf` for mobile server endpoint

### Handler Files (2 files)
7. **pkg/internal/apis/handlers/credentials_deeplink_handlers.go**
   - Replaced string concatenation for `/api/get-deeplink` path

8. **pkg/internal/apis/handlers/verifiers_deeplink_handlers.go**
   - Replaced string concatenation for `/api/get-deeplink` path

### Activity Files (1 file)
9. **pkg/workflowengine/activities/credentialsissuer.go**
   - Replaced string concatenation for `.well-known` endpoint URLs (2 instances)

### Other Files (2 files)
10. **pkg/generate_client/generate_client.go**
    - Replaced `path.Join` with `url.JoinPath` for URL path segments

11. **pkg/credential_issuer/workflow/shared.go**
    - Simplified constant URL (removed unnecessary string concatenation)

## Patterns Found and Fixed

### ✅ Fixed Patterns
1. **String concatenation with `+` and `"/"`**
   - Found: 15+ instances
   - Fixed: All URL-building instances

2. **`fmt.Sprintf` for URL paths**
   - Found: 10+ instances
   - Fixed: All URL-building instances

3. **`path.Join` for URLs**
   - Found: 1 instance
   - Fixed: Replaced with `url.JoinPath`

### ⚠️ Patterns That Are Acceptable (Not Changed)
1. **File path building** - Using `path.Join` or `filepath.Join` for file system paths (not URLs)
2. **URL path building for canonified paths** - `pkg/internal/canonify/resolver.go` builds URL paths (not full URLs), which is acceptable
3. **URL normalization** - Adding scheme prefix (`"https://" + cleanURL`) is acceptable when properly validated
4. **Query parameter building** - Using `url.Query()` and `query.Set()` is the correct approach
5. **Deep link building** - Custom scheme URLs (like `eudi-openid4vp://`) may use `fmt.Sprintf` with proper encoding

## Additional URL Building Patterns to Watch For

### Patterns Not Yet Found (But Should Be Checked)
1. **Template-based URL building**
   ```go
   // Watch for:
   template.Must(template.New("url").Parse("{{.Base}}/api/{{.Path}}"))
   ```

2. **Regex-based URL manipulation**
   ```go
   // Watch for:
   re.ReplaceAllString(url, ...)
   ```

3. **Multiple chained operations**
   ```go
   // Watch for:
   url = strings.TrimSuffix(url, "/") + "/newpath"
   ```

4. **URL building in loops**
   ```go
   // Watch for:
   for _, segment := range segments {
       url += "/" + segment
   }
   ```

5. **Conditional URL building**
   ```go
   // Watch for:
   if condition {
       url = base + "/path1"
   } else {
       url = base + "/path2"
   }
   ```

## Tools Created

1. **Search Script**: `scripts/find_manual_urls.sh`
   - Enhanced script with 12 different pattern searches
   - Can be run anytime to find new manual URL building

2. **Documentation**: `docs/URL_BUILDING_PATTERNS.md`
   - Comprehensive guide on safe vs unsafe patterns
   - Examples and migration checklist

## Recommendations

1. **Run the search script regularly** (e.g., in CI/CD or before releases)
2. **Code review checklist**: Check for manual URL building in PRs
3. **Linter rule**: Consider adding a custom linter rule to catch these patterns
4. **Team training**: Share the URL building patterns guide with the team

## Testing

All changes have been:
- ✅ Tested for linter errors (none found)
- ✅ Verified to maintain existing functionality
- ✅ Using proper URL encoding and path joining

## Future Improvements

1. Consider creating a `BuildURLWithQuery()` helper function for common query parameter patterns
2. Consider creating a `NormalizeURL()` helper for scheme normalization
3. Add unit tests for edge cases (trailing slashes, special characters, etc.)

