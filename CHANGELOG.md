## v0.1.0 (2025-09-26)

This release introduces new features for controlling API output and standardizes the logging system. The `gemapi request` command now supports generation parameters, allowing for fine-grained control over the Gemini API's output. Users can specify temperature, top-p, top-k, and maximum output tokens via new command-line flags to influence the creativity and length of responses (9c2fc0d).

The logging infrastructure has been refactored to align with the `grove-core` ecosystem (3c6b23d). The custom pretty logger has been replaced with a wrapper around the `grove-core` logger, enabling a dual-logging system that provides both human-readable UI feedback and structured data logs (185c115, 7c99199). This change preserves all Gemini-specific logging functionality while standardizing log output across Grove tools. Housekeeping updates include improvements to the `.gitignore` file (50e3fff).

### Features

- Add generation parameters to control output length and style (9c2fc0d)
- Use new grove core logging (3c6b23d)

### Refactoring

- Replace custom pretty logger with grove-core wrapper (185c115)
- Restore pretty logging with grove-core integration (7c99199)

### Chores

- update .gitignore rules (50e3fff)

### File Changes

```
 .gitignore            |   7 ++
 CLAUDE.md             |  30 +++++
 cmd/request.go        |  25 ++++
 pkg/gemini/cache.go   |   6 +-
 pkg/gemini/client.go  |  78 +++++++-----
 pkg/gemini/logger.go  |   8 ++
 pkg/gemini/request.go |  12 +-
 pkg/gemini/upload.go  |   2 +-
 pkg/pretty/logger.go  | 335 +++++++++++++++++++++++++++++++-------------------
 9 files changed, 343 insertions(+), 160 deletions(-)
```

## v0.1.0 (2025-09-26)

This release introduces new features for controlling API output and standardizes the logging system. The `gemapi request` command now supports generation parameters, allowing for fine-grained control over the Gemini API's output. Users can specify temperature, top-p, top-k, and maximum output tokens via new command-line flags to influence the creativity and length of responses (9c2fc0d).

The logging infrastructure has been refactored to align with the `grove-core` ecosystem (3c6b23d). The custom pretty logger has been replaced with a wrapper around the `grove-core` logger, enabling a dual-logging system that provides both human-readable UI feedback and structured data logs (185c115, 7c99199). This change preserves all Gemini-specific logging functionality while standardizing log output across Grove tools. Housekeeping updates include improvements to the `.gitignore` file (50e3fff).

### Features

- Add generation parameters to control output length and style (9c2fc0d)
- Use new grove core logging (3c6b23d)

### Refactoring

- Replace custom pretty logger with grove-core wrapper (185c115)
- Restore pretty logging with grove-core integration (7c99199)

### Chores

- update .gitignore rules (50e3fff)

### File Changes

```
 .gitignore            |   7 ++
 CLAUDE.md             |  30 +++++
 cmd/request.go        |  25 ++++
 pkg/gemini/cache.go   |   6 +-
 pkg/gemini/client.go  |  78 +++++++-----
 pkg/gemini/logger.go  |   8 ++
 pkg/gemini/request.go |  12 +-
 pkg/gemini/upload.go  |   2 +-
 pkg/pretty/logger.go  | 335 +++++++++++++++++++++++++++++++-------------------
 9 files changed, 343 insertions(+), 160 deletions(-)
```

## v0.0.14 (2025-09-17)

### Chores

* bump dependencies
* update Grove dependencies to latest versions

## v0.0.12 (2025-09-12)

### Bug Fixes

* disable Gemini cache to reduce unexpected costs

## v0.0.11 (2025-09-12)

### Features

* implement opt-in cache safety with @enable-cache directive

### Bug Fixes

* address code review findings for cache-safety
* disable cache

### Chores

* **deps:** bump dependencies
* remove indirect deps
* delete go.work

## v0.0.10 (2025-09-06)

### Features

* add comprehensive cache analytics and insights
* add regeneration counter to track cache recreations
* add interactive TUI for gemapi cache management
* enhance cache management with API integration and usage tracking

### Chores

* **deps:** sync Grove dependencies to latest versions

## v0.0.9 (2025-08-29)

### Features

* add flexible API key configuration with explicit passing support

### Chores

* **deps:** sync Grove dependencies to latest versions
* **deps:** sync Grove dependencies to latest versions

### Bug Fixes

* **tests:** update API key config tests to use request command and isolate environment

## v0.0.8 (2025-08-28)

### Chores

* **deps:** sync Grove dependencies to latest versions
* **deps:** sync Grove dependencies to latest versions

### Features

* add debug logging for Gemini API requests

### Bug Fixes

* move debug logging before file upload to ensure logging on failures
* clarify file attachment logging message
* implement file deduplication and proper prompt file handling

## v0.0.7 (2025-08-27)

### Chores

* **deps:** sync Grove dependencies to latest versions
* update readme

## v0.0.6 (2025-08-26)

### Bug Fixes

* include prompt file contents in API requests

### Features

* move file list display before API request
* add prompt file tracking and user token counting

## v0.0.5 (2025-08-25)

### Chores

* **deps:** sync Grove dependencies to latest versions
* **deps:** sync Grove dependencies to latest versions

## v0.0.4 (2025-08-25)

### Bug Fixes

* typo

## v0.0.3 (2025-08-25)

### Chores

* **deps:** sync Grove dependencies to latest versions
* bump dependencies

### Features

* expose request functionality as reusable Go package

### Bug Fixes

* disable lefs
* disable linting
* disable lfs

## v0.0.2 (2025-08-25)

### Bug Fixes

* improve logging and various cache issues

### Features

* add --use-cache flag to request command for explicit cache selection
* enhance cache management with new subcommands and improvements
* add cache logs/metrics/billing query commands
* add disable cache directive
* move gemini client from grove-flow to this package

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial implementation of grove-gemini
- Basic command structure
- E2E test framework
