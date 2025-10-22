# Experimental Features

This section covers features under active development. Their interfaces and behavior are subject to change.

## Context Caching

The context caching feature uses the Gemini Caching API to store "cold context" files between requests.

> **⚠️ CRITICAL WARNING: Risk of Substantial Unexpected Charges**
>
> The Gemini Caching API is a billable service. Misuse or misconfiguration of this feature can lead to substantial and unexpected charges on your Google Cloud account.
>
> The primary risk comes from frequent cache invalidation. If "cold context" files change often or the cache time-to-live (TTL) is misconfigured, `gemapi` will repeatedly create new caches. This action incurs costs for both the cache creation API calls and the storage of the cached content.
>
> **This feature is NOT recommended for general use until further stabilized.** Use it only if you understand the cost implications and are actively monitoring your billing.

The caching functionality is enabled by adding an `@enable-cache` directive to your `.grove/rules` file. The associated command-line flags (`--no-cache`, `--recache`, `--use-cache`, `--cache-ttl`) and other directives (`@freeze-cache`, `@no-expire`) are also considered experimental.

## Observability Integration

Integration with `grove-hooks` for session tracking and performance monitoring is in development. The event schemas and integration points may change in future releases.