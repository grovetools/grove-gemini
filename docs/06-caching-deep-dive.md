# Caching Deep Dive

This guide provides an in-depth explanation of `gemapi`'s caching functionality, including how to enable it, manage cache lifecycles, and use advanced features for optimal performance and cost management.

## Opt-In Mechanism

Caching in `gemapi` is **disabled by default** to prevent unexpected costs and ensure users make deliberate decisions about cache usage. To enable caching, you must explicitly add the `@enable-cache` directive to your `.grove/rules` file.

### Enabling Caching

Add the `@enable-cache` directive as a standalone line in your `.grove/rules` file:

```
@enable-cache
**/*.go
**/*.md
!**/*_test.go
```

**Important Rules:**
- The `@enable-cache` directive must be on its own line
- It cannot have comments on the same line
- Leading/trailing whitespace is acceptable
- If commented out (with `#`), caching remains disabled

### Verification

You can verify caching is enabled by running a request. If caching is disabled, you'll see:

```
❌ Caching disabled
ℹ️ To enable caching, add @enable-cache to your .grove/rules file
```

## Cache Lifecycle

Understanding the cache lifecycle helps you optimize usage and troubleshoot issues.

### 1. Cache Creation

When you make a request with caching enabled:

1. **Content Analysis**: `gemapi` analyzes cold context files and generates a hash
2. **Cache Check**: System checks if a valid cache exists for this content
3. **Cache Creation**: If no valid cache exists, a new one is created
4. **Cost Confirmation**: For expensive caches, you'll be prompted to confirm

```bash
# Example cache creation flow
$ gemapi request "Analyze the codebase architecture"

📊 Cache Creation Required
   Content Hash: a1b2c3d4...
   Estimated Cost: $0.45
   Files: 127 files, 45,000 tokens

❓ Create cache for $0.45? (y/N): y

✅ Cache created: project-main-a1b2c3d4
   Cache ID: cachedContent/abc123
   Expires: 2024-10-01 15:30:00 UTC
```

### 2. Cache Usage

Once created, caches are automatically used for subsequent requests with the same cold context:

```bash
$ gemapi request "What design patterns are used in this codebase?"

✅ Using existing cache: project-main-a1b2c3d4
   Cache Hit: 45,000 tokens cached
   Cost Savings: $0.36 per request
```

### 3. Cache Invalidation

Caches are automatically invalidated when cold context files change:

```bash
# After modifying a file in cold context
$ gemapi request "Review the updated code structure"

⚠️ Cache invalidated: project-main-a1b2c3d4
   Reason: File content changed (3 files modified)
   Creating new cache...
```

### 4. Cache Expiration

Caches have configurable time-to-live (TTL) periods. When a cache expires, it's automatically recreated on the next request:

```bash
$ gemapi request "Analyze the codebase"

⏰ Cache expired: project-main-a1b2c3d4
   Expired: 2024-09-30 10:15:00 UTC
   Creating new cache...
```

## Cache Directives

`gemapi` supports several special directives in `.grove/rules` files to control cache behavior:

### @freeze-cache

Prevents cache invalidation from file changes. Useful for expensive caches during development:

```
@enable-cache
@freeze-cache
**/*.go
**/*.md
```

When active:
```bash
$ gemapi request "Analyze the code"

🔒 Cache frozen by @freeze-cache directive
   Using cache despite file changes
   To update cache, remove @freeze-cache and use --recache
```

### @no-expire

Creates a cache with the maximum possible TTL (effectively permanent):

```
@enable-cache
@no-expire
**/*.go
**/*.md
```

Effects:
- Cache will not expire due to time
- Still subject to invalidation from file changes (unless @freeze-cache is also used)
- Useful for stable, expensive-to-cache content

### @expire-time

Specifies a custom TTL for the cache:

```
@enable-cache
@expire-time 48h
**/*.go
**/*.md
```

Supported formats:
- `24h` (24 hours)
- `7d` (7 days)  
- `2h30m` (2 hours 30 minutes)
- `168h` (1 week in hours)

Example with custom TTL:
```bash
$ gemapi request "Review the codebase"

✅ Cache created with custom TTL
   Duration: 48h (expires 2024-10-02 14:30:00 UTC)
   Directive: @expire-time 48h
```

### Combining Directives

You can combine directives for specific behaviors:

```
@enable-cache
@freeze-cache
@no-expire
**/*.go
**/*.md
```

This creates a permanent cache that ignores file changes - useful for expensive reference materials.

## Manual Cache Control

### Command-Line Flags

#### --recache

Forces creation of a new cache, ignoring existing valid caches:

```bash
# Force cache recreation
gemapi request --recache "Analyze the updated codebase"

🔄 Forced cache recreation requested
   Ignoring existing cache: project-main-a1b2c3d4
   Creating new cache...
```

#### --use-cache

Specifies a particular cache to use by name:

```bash
# Use a specific cache
gemapi request --use-cache project-main-a1b2c3d4 "Review the code"

✅ Using specified cache: project-main-a1b2c3d4
   Cache ID: cachedContent/abc123
   Status: Valid (expires in 2d 14h)
```

#### --no-cache

Disables caching for a single request:

```bash
# Skip caching entirely
gemapi request --no-cache "Quick code review"

❌ Caching disabled by --no-cache flag
   Sending cold context directly (45,000 tokens)
```

## Cache Management CLI

The `gemapi cache` command provides comprehensive cache management:

### Listing Caches

```bash
# List all caches
gemapi cache list

# Example output:
┌─────────────────────┬──────────────┬─────────┬─────────────────────┬──────────┐
│ Name                │ Status       │ Tokens  │ Expires             │ Usage    │
├─────────────────────┼──────────────┼─────────┼─────────────────────┼──────────┤
│ project-main-a1b2c3 │ Active       │ 45,000  │ 2024-10-01 15:30:00 │ 23 hits  │
│ project-feat-b2c3d4 │ Expired      │ 32,000  │ 2024-09-28 10:15:00 │ 8 hits   │
│ docs-update-c3d4e5  │ Invalid      │ 12,000  │ 2024-10-02 09:00:00 │ 3 hits   │
└─────────────────────┴──────────────┴─────────┴─────────────────────┴──────────┘
```

### Inspecting Caches

```bash
# Detailed cache information
gemapi cache inspect project-main-a1b2c3d4

# Example output:
Cache: project-main-a1b2c3d4
├─ Cache ID: cachedContent/abc123def456
├─ Status: Active
├─ Model: gemini-1.5-pro-latest
├─ Created: 2024-09-26 10:30:00 UTC
├─ Expires: 2024-10-01 15:30:00 UTC (in 4d 14h)
├─ Token Count: 45,127 tokens
├─ Repository: grove-gemini (main branch)
├─ Usage Statistics:
│  ├─ Total Queries: 23
│  ├─ Last Used: 2024-09-30 14:15:00 UTC
│  ├─ Cache Hits: 1,038,921 tokens served
│  ├─ Tokens Saved: 925,483 tokens
│  └─ Average Hit Rate: 89.1%
└─ Files (127 total):
   ├─ cmd/root.go (832 bytes, hash: a1b2c3...)
   ├─ pkg/gemini/client.go (4,521 bytes, hash: b2c3d4...)
   └─ [125 more files...]
```

### Clearing Caches

```bash
# Clear a specific cache
gemapi cache clear project-main-a1b2c3d4

# Clear all caches for current project
gemapi cache clear --all

# Prune expired caches
gemapi cache prune
```

## Interactive TUI

The `gemapi cache tui` provides a rich terminal interface for cache management:

```bash
gemapi cache tui
```

### TUI Features

**Cache List View:**
- Browse all caches with sorting options
- Real-time status updates
- Color-coded status indicators
- Quick filtering and search

**Cache Inspection:**
- Detailed cache information
- File-by-file content breakdown
- Usage analytics and trends
- Performance metrics

**Cache Management:**
- Delete caches with confirmation
- Bulk operations on multiple caches
- Export cache information
- Generate usage reports

**Navigation:**
- `↑/↓` or `j/k`: Navigate lists
- `Enter`: Inspect selected cache
- `d`: Delete cache (with confirmation)
- `r`: Refresh cache status
- `q`: Quit

### TUI Analytics View

The analytics view provides insights into cache usage patterns:

```
Cache Analytics
├─ Total Caches: 12
├─ Active Caches: 8
├─ Total Token Savings: 2,847,329 tokens
├─ Estimated Cost Savings: $22.78
├─ Average Cache Hit Rate: 84.3%
├─ Most Used Cache: project-main-a1b2c3d4 (89 queries)
└─ Cache Efficiency Trend: ↗ +12.4% this week
```

## Best Practices for Caching

### Cost Management
1. **Monitor Cache Creation**: Large caches can be expensive to create
2. **Use Appropriate TTLs**: Longer TTLs for stable content, shorter for active development
3. **Leverage @freeze-cache**: During heavy development to avoid frequent recreations

### Performance Optimization
1. **Cache Stable Content**: Libraries, documentation, and core architecture
2. **Keep Hot Context Small**: Frequently changing files should stay in hot context
3. **Use Specific Caches**: Use `--use-cache` for specific analysis scenarios

### Development Workflow
1. **Enable Caching Early**: Set up caching before your codebase grows large
2. **Monitor Invalidation**: Watch for unexpected cache invalidations
3. **Regular Pruning**: Use `gemapi cache prune` to clean up expired caches

The caching system in `gemapi` is designed to provide significant cost and performance benefits for large codebases while maintaining automatic management and data freshness. Understanding these concepts will help you optimize your usage patterns and minimize API costs.