# Core Concepts

This guide explains the fundamental concepts that you need to understand to use `gemapi` effectively. These concepts form the foundation of how `gemapi` manages context, optimizes performance through caching, and provides comprehensive observability.

## Integration with grove-context

`gemapi` leverages the Grove ecosystem's `grove-context` system to intelligently manage project context. The core of this integration is the `.grove/rules` file, which defines how your project's codebase is partitioned and included in API requests.

The `.grove/rules` file uses glob patterns to specify which files should be included in different contexts:

```
# Example .grove/rules file
**/*.go          # Include all Go files
**/*.md          # Include all Markdown files
!**/vendor/**    # Exclude vendor directory
!**/*_test.go    # Exclude test files
```

When you run `gemapi request`, the system automatically:

1. Reads your `.grove/rules` file
2. Scans your project to find matching files
3. Partitions them into "hot" and "cold" context based on size and change frequency
4. Includes the appropriate context in your API request

This integration ensures that the Gemini API has relevant context about your codebase without you having to manually specify files for each request.

## Hot vs. Cold Context

One of `gemapi`'s key innovations is its intelligent partitioning of project context into "hot" and "cold" categories:

### Cold Context (`.grove/cached-context`)

Cold context consists of:
- Large files that change infrequently
- Stable architectural components
- Libraries and dependencies
- Documentation that rarely changes

These files are ideal candidates for the Gemini Caching API because:
- They provide important context but don't change often
- They can be expensive to include in every request due to their size
- Caching them significantly reduces token costs and improves latency

Cold context files are stored in `.grove/cached-context` and can be uploaded to Google's caching service for reuse across multiple requests.

### Hot Context (`.grove/context`)

Hot context includes:
- Files that change frequently during development
- Small files that are inexpensive to include directly
- Files that have been recently modified
- Dynamic content that varies between requests

Hot context files are stored in `.grove/context` and are included directly with each API request. This ensures that the most current version of frequently-changing files is always used.

### Context Flow Diagram

```
Your Project Files
       ↓
.grove/rules (filtering)
       ↓
grove-context (analysis)
       ↓
    ┌─────────────────┐
    │                 │
    ▼                 ▼
Cold Context      Hot Context
(large/stable)    (small/dynamic)
    │                 │
    ▼                 │
Gemini Cache          │
(reusable)            │
    │                 │
    └────────┬────────┘
             ▼
      Gemini API Request
```

## The Caching Layer

`gemapi`'s caching system is designed to optimize both cost and performance when working with large codebases:

### Opt-In Design

Caching is **disabled by default** to prevent unexpected costs. To enable caching, you must explicitly add the `@enable-cache` directive to your `.grove/rules` file:

```
@enable-cache
**/*.go
**/*.md
!**/*_test.go
```

### How Caching Works

1. **Cache Creation**: When enabled, `gemapi` uploads cold context files to Google's Gemini Caching API
2. **Cache Reuse**: Subsequent requests reference the cached content instead of re-uploading
3. **Automatic Invalidation**: Caches are automatically invalidated when cold context files change
4. **Cost Optimization**: Large, stable files are cached while dynamic files are sent directly

### Cache Benefits

- **Reduced Token Costs**: Cached content doesn't count toward input token limits for each request
- **Improved Latency**: Less data to upload means faster request processing
- **Automatic Management**: No manual cache invalidation needed - the system tracks file changes

## Observability Data Sources

`gemapi` provides comprehensive observability through its `query` command, which pulls data from multiple sources to give you complete visibility into your API usage:

### Local Logs
- **Source**: Local JSONL files stored on your machine
- **Data**: Request details, token usage, costs, latency, git context
- **Access**: `gemapi query local`
- **Scope**: All requests made from your local machine

### Google Cloud Monitoring
- **Source**: Google Cloud Monitoring metrics
- **Data**: Request counts, error rates, aggregated statistics
- **Access**: `gemapi query metrics`
- **Scope**: Project-wide metrics across all users and services

### Google Cloud Logging
- **Source**: Google Cloud Logging service
- **Data**: Detailed token usage information from the Gemini API
- **Access**: `gemapi query tokens`
- **Scope**: Token-level details for billing analysis

### BigQuery Billing Export
- **Source**: BigQuery billing export tables
- **Data**: SKU-level cost breakdown, detailed billing information
- **Access**: `gemapi query billing`
- **Scope**: Complete cost analysis including Gemini API and caching costs
- **Prerequisite**: Requires BigQuery billing export to be configured in your GCP account

### Observability Flow

```
gemapi request
      ↓
   [Local Log] ──────────→ query local
      ↓
  Gemini API
      ↓
   [GCP Logs] ──────────→ query tokens
      ↓
 [GCP Metrics] ─────────→ query metrics
      ↓
[BigQuery Billing] ────→ query billing
```

Each data source provides different levels of detail and scope, allowing you to analyze usage patterns from multiple perspectives:

- **Development**: Use `query local` for immediate feedback on your requests
- **Operations**: Use `query metrics` for service-level monitoring
- **Financial**: Use `query billing` for detailed cost analysis
- **Optimization**: Use `query tokens` for token usage optimization

Understanding these core concepts will help you use `gemapi` effectively, whether you're optimizing for cost, performance, or gaining insights into your API usage patterns.