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