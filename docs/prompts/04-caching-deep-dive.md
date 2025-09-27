# Caching Deep Dive Documentation

You are writing an in-depth guide to the caching functionality in `gemapi`.

## Task
Explain the caching mechanism in detail. Cover the following topics:
1.  **Opt-In Mechanism**: Explain that caching is disabled by default and must be enabled with the `@enable-cache` directive in `.grove/rules`.
2.  **Cache Lifecycle**: Describe how a cache is created, used, invalidated, and expired.
3.  **Cache Invalidation**: Explain that caches are automatically invalidated when the content of the "cold context" files changes.
4.  **Cache Directives**: Document the special directives that can be placed in `.grove/rules`:
    - `@freeze-cache`: To prevent invalidation from file changes.
    - `@no-expire`: To create a cache with the maximum possible TTL.
    - `@expire-time <duration>`: To specify a custom TTL.
5.  **Manual Cache Control**: Explain the `--recache` and `--use-cache` flags for `gemapi request`.
6.  **Cache Management CLI**: Briefly re-introduce the `gemapi cache` subcommands as the primary way to interact with and manage caches.
7.  **Interactive TUI**: Describe the `gemapi cache tui` and what users can do with it (view, inspect, delete caches).

## Output Format
- Use clear headings for each topic.
- Provide small examples of `.grove/rules` files demonstrating the directives.
- Use command-line examples for flags and `gemapi cache` commands.
- Reference `pkg/gemini/cache.go` and `cmd/cache_tui.go` for implementation details.