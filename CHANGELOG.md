## v0.6.0 (2026-02-02)

This release improves configuration management by migrating to TOML format (44bba64) and adopting XDG-compliant paths for GCP settings (d748c8d) and logs (6d25b22). CLI output handling has been fixed to route logging to stderr (a0a7e5c), facilitating easier output piping. Additionally, the project has been updated with an MIT License (79daf3d), restored release workflows (04ad7c9), and migrated dependencies to the grovetools namespace (8ccdcc8).

### Features
- Add configuration/readme update (b8ab33b)

### Bug Fixes
- Use XDG config path for GCP configuration (d748c8d)
- Route logging to stderr so stdout can be piped (a0a7e5c)
- Update VERSION_PKG to grovetools/core path (46657ce)

### Code Refactoring
- Migrate logging to XDG-compliant paths (6d25b22)
- Update docgen title to match package name (03f54f1)

### Documentation
- Add concept lookup instructions to CLAUDE.md (c8b339d)

### Chores
- Add MIT License (79daf3d)
- Migrate grove.yml to grove.toml (44bba64)
- Update go.mod for grovetools migration (8ccdcc8)
- Gemapi replacements, update readme (de59582)

### CI
- Restore release workflow (04ad7c9)

### File Changes
```
 .cx/docs.rules                   |  16 ++++
 .github/workflows/release.yml    | 114 ++++++++++-------------------
 .gitignore                       |   2 +-
 CHANGELOG.md                     |   6 +-
 CLAUDE.md                        |  17 ++++-
 LICENSE                          |  21 ++++++
 Makefile                         |   8 +-
 README.md                        |  65 +++++++----------
 cmd/cache.go                     |   2 +-
 cmd/config.go                    |   4 +-
 cmd/count_tokens.go              |   8 +-
 cmd/query_billing.go             |   6 +-
 cmd/query_dashboard.go           |   6 +-
 cmd/query_explore.go             |   2 +-
 cmd/query_metrics.go             |   2 +-
 cmd/query_tokens.go              |   2 +-
 cmd/request.go                   |  28 +++----
 cmd/root.go                      |   2 +-
 docs/01-overview.md              |  58 ++++++---------
 docs/02-examples.md              |  34 ++++-----
 docs/03-configuration.md         |  15 ++++
 docs/03-experimental.md          |   2 +-
 docs/04-configuration.md         |  18 ++---
 docs/05-command-reference.md     |  86 +++++++++++-----------
 docs/docgen.config.yml           |   4 +-
 gemini.schema.json               |   2 +-
 go.mod                           |  10 ++-
 go.sum                           |  47 ++++++++++--
 grove.toml                       |  11 +++
 grove.yml                        |  10 ---
 main.go                          |   4 +-
 pkg/config/gcp.go                |  12 +--
 pkg/docs/docs.json               | 154 ++-------------------------------------
 pkg/gemini/request.go            |   2 +-
 pkg/logging/query_log.go         |  16 ++--
 tests/e2e/scenarios_query_tui.go |  16 ++--
 tests/e2e/test_utils.go          |  16 ++--
 37 files changed, 368 insertions(+), 460 deletions(-)
```

## v0.2.1-nightly.655550b (2025-10-03)

## v0.2.0 (2025-10-01)

This release introduces a comprehensive documentation overhaul, establishing a standardized structure with new sections for an overview, examples, experimental features, configuration, and a complete command reference (d8fd7ac, b56e0f0, e557c21). The documentation content has been refined to be more succinct and aligned with the Grove ecosystem's philosophy (03b38be, c3a426e).

Tooling for documentation has been significantly improved by integrating `grove-docgen` to automate the generation of the `README.md` file, which now includes a Table of Contents (887ed39). The configuration for `docgen` has also been standardized for consistency (ff6a667).

The release process is now more robust, with the release workflow updated to extract release notes directly from `CHANGELOG.md` (be1ef9f). Additionally, the CI workflow has been refined to prevent unnecessary executions while maintaining valid syntax (7848336) and to remove redundant test runs from the release process (cf478ac).

### Features

- Add comprehensive project documentation (c3a426e)
- Add automated Table of Contents generation to README (887ed39)
- Update release workflow to extract notes from CHANGELOG.md (be1ef9f)

### Bug Fixes

- Update CI workflow to use 'branches: [ none ]' to prevent execution (7848336)
- Clean up README.md.tpl template format (1655f65)
- Remove old documentation files (1dcf74d)

### Build

- Remove redundant tests from release workflow (cf478ac)

### Refactoring

- Standardize docgen.config.yml key order and settings (ff6a667)

### Chores

- Temporarily disable CI workflow (0253f6d)
- Standardize documentation filenames to DD-name.md convention (94f2902)

### File Changes

```
 .github/workflows/ci.yml             |   4 +-
 .github/workflows/release.yml        |  13 +-
 Makefile                             |   8 +-
 README.md                            | 190 ++---------
 docs/01-overview.md                  |  47 +++
 docs/02-examples.md                  | 161 ++++++++++
 docs/03-experimental.md              |  21 ++
 docs/04-configuration.md             |  96 ++++++
 docs/05-command-reference.md         | 246 +++++++++++++++
 docs/README.md.tpl                   |   7 +
 docs/docgen.config.yml               |  45 +++
 docs/docs.rules                      |   1 +
 docs/images/grove-gemini-readme.svg  | 592 +++++++++++++++++++++++++++++++++++
 docs/prompts/01-overview.md          |  31 ++
 docs/prompts/02-examples.md          |  24 ++
 docs/prompts/03-experimental.md      |  18 ++
 docs/prompts/04-configuration.md     |  23 ++
 docs/prompts/05-command-reference.md |  27 ++
 pkg/docs/docs.json                   | 159 ++++++++++
 19 files changed, 1548 insertions(+), 165 deletions(-)
```

## v0.1.0 (2025-09-26)

This release introduces new features for controlling API output and standardizes the logging system. The `grove-gemini request` command now supports generation parameters, allowing for fine-grained control over the Gemini API's output. Users can specify temperature, top-p, top-k, and maximum output tokens via new command-line flags to influence the creativity and length of responses (9c2fc0d).

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

This release introduces new features for controlling API output and standardizes the logging system. The `grove-gemini request` command now supports generation parameters, allowing for fine-grained control over the Gemini API's output. Users can specify temperature, top-p, top-k, and maximum output tokens via new command-line flags to influence the creativity and length of responses (9c2fc0d).

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
* add interactive TUI for grove-gemini cache management
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
