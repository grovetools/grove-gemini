# Experimental Features

This section covers features in `grove-gemini` that are currently experimental. They are subject to change, may have limitations, and should be used with caution.

## Context Caching

The context caching feature leverages the Gemini Caching API to reduce costs and latency on subsequent requests with large, unchanging "cold context" files. While this can be effective, it is an experimental feature with significant risks.

> **⚠️ CRITICAL WARNING: Risk of Substantial Unexpected Charges**
>
> The Gemini Caching API is a billable service. Misuse or misconfiguration of this feature can lead to substantial and unexpected charges on your Google Cloud account.
>
> The primary risk comes from frequent cache invalidation. If your "cold context" files change often or your cache TTL is too short, `gemapi` will repeatedly create new caches, incurring costs for both the cache creation API calls and the storage of the cached content.
>
> **This feature is NOT recommended for general use until it is further stabilized.** Use it only if you fully understand the cost implications and are actively monitoring your billing.

The caching functionality is enabled by adding an `@enable-cache` directive to your `.grove/rules` file. The behavior can be controlled with command-line flags (`--no-cache`, `--recache`, `--use-cache`, `--cache-ttl`) and other directives (`@freeze-cache`, `@no-expire`). These controls are also considered experimental.

## Observability Integration

Integration with the broader Grove ecosystem for observability, such as emitting detailed events to `grove-hooks` for session tracking and performance monitoring, is currently under development. The event schemas and integration points are subject to change in future releases.