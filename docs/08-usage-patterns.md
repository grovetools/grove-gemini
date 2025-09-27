# Usage Patterns

This guide provides practical examples and common usage patterns for the `gemapi` CLI, demonstrating how to leverage its features effectively in real-world scenarios.

## Simple Q&A

The most basic usage pattern involves asking questions about text or code without complex context management.

### Basic Text Generation

```bash
# Simple prompt from command line
gemapi request "Explain the concept of microservices in 3 paragraphs"

# Using a prompt file for longer content
echo "Write a technical blog post about GraphQL vs REST APIs, focusing on performance implications and use cases." > prompt.md
gemapi request -f prompt.md

# Specifying output location
gemapi request -f prompt.md -o blog-post.md

# Using a different model for cost optimization
gemapi request -m gemini-1.5-flash-latest "Summarize this text" -o summary.txt
```

### Quick Code Analysis

```bash
# Analyze a single file
gemapi request "Review this function for potential improvements" --context-files main.go

# Generate documentation for code
gemapi request "Generate JSDoc comments for all functions in this file" --context-files utils.js -o documented-utils.js

# Code conversion
gemapi request "Convert this Python function to TypeScript" --context-files legacy.py -o converted.ts
```

## Codebase Analysis

Leveraging `grove-context` integration for comprehensive codebase understanding.

### Setting Up Context

First, create a `.grove/rules` file to define which files should be included:

```bash
# Create basic rules for a Go project
cat > .grove/rules << 'EOF'
**/*.go
**/*.md
!**/*_test.go
!**/vendor/**
!**/node_modules/**
EOF
```

### Architecture Analysis

```bash
# High-level architecture review
gemapi request "Analyze the overall architecture of this codebase. Identify the main components, their relationships, and any architectural patterns used."

# Identify code smells and improvements
gemapi request "Review this codebase for potential code smells, performance issues, and areas for improvement. Provide specific recommendations."

# Documentation generation
gemapi request "Generate comprehensive documentation for this project, including setup instructions, API reference, and usage examples." -o PROJECT_DOCS.md
```

### Specific Analysis Tasks

```bash
# Security review
gemapi request "Perform a security analysis of this codebase. Identify potential vulnerabilities, security best practices violations, and recommend fixes."

# Performance analysis
gemapi request "Analyze this codebase for performance bottlenecks. Focus on database queries, API calls, and algorithm efficiency."

# Testing strategy
gemapi request "Review the current testing approach and suggest improvements. Identify areas that need better test coverage."
```

## Iterative Development with Caching

Demonstrate a workflow that benefits from caching for large codebases.

### Initial Setup

```bash
# Enable caching for large, stable codebases
cat > .grove/rules << 'EOF'
@enable-cache
**/*.go
**/*.ts
**/*.py
**/*.md
!**/*_test.go
!**/dist/**
!**/build/**
EOF
```

### First Request (Cache Creation)

```bash
# Initial analysis - creates cache
gemapi request "Provide a comprehensive overview of this codebase architecture"

# Output shows cache creation:
# 📊 Cache Creation Required
#    Content Hash: a1b2c3d4...
#    Estimated Cost: $0.45
#    Files: 127 files, 45,000 tokens
# ❓ Create cache for $0.45? (y/N): y
# ✅ Cache created: project-main-a1b2c3d4
```

### Subsequent Requests (Cache Usage)

```bash
# Follow-up questions use the cache automatically
gemapi request "What design patterns are implemented in the authentication module?"

# Output shows cache usage:
# ✅ Using existing cache: project-main-a1b2c3d4
#    Cache Hit: 45,000 tokens cached
#    Cost Savings: $0.36 per request

# More detailed analysis
gemapi request "Suggest refactoring opportunities for the data access layer"

# Performance optimization questions
gemapi request "Identify bottlenecks in the API endpoints and suggest optimizations"
```

### Development Iteration

```bash
# During active development, freeze cache to avoid frequent recreation
cat > .grove/rules << 'EOF'
@enable-cache
@freeze-cache
**/*.go
**/*.ts
**/*.py
**/*.md
!**/*_test.go
EOF

# Continue asking questions without cache invalidation
gemapi request "How would adding a message queue affect the current architecture?"

# When ready to update cache with new changes
sed -i '/^@freeze-cache$/d' .grove/rules  # Remove freeze directive
gemapi request --recache "Analyze the updated codebase after recent changes"
```

## Cost and Usage Monitoring

Comprehensive workflow for tracking and optimizing API usage costs.

### Daily Development Monitoring

```bash
# Quick check of today's usage
gemapi query local --hours 8

# Check for any errors during development
gemapi query local --errors --hours 4

# Monitor token usage efficiency
gemapi query tokens --hours 24 --project-id your-project
```

### Weekly Cost Review

```bash
# Comprehensive local usage analysis
gemapi query local --hours 168 --limit 1000 > weekly-usage.txt

# Project-wide metrics
gemapi query metrics --project-id your-project --hours 168

# Detailed cost breakdown (requires BigQuery billing export)
gemapi query billing \
  --project-id your-project \
  --dataset-id billing_export \
  --table-id gcp_billing_export_v1_ABCDEF_123456 \
  --days 7
```

### Cost Optimization Workflow

```bash
# 1. Identify high-cost requests
gemapi query local --hours 168 | grep -E '\$[0-9]+\.[5-9][0-9]'  # Find requests costing > $0.50

# 2. Analyze token usage patterns
gemapi query tokens --project-id your-project --hours 168

# 3. Review cache effectiveness
gemapi cache list

# 4. Optimize based on findings
# - Use cheaper models for simple tasks
gemapi request -m gemini-1.5-flash-latest "Simple text processing task"

# - Enable caching for repeated analysis
echo "@enable-cache" | cat - .grove/rules > temp && mv temp .grove/rules

# 5. Validate improvements
gemapi query billing --days 1  # Check recent costs
```

## Pre-flight Checks

Using token counting to estimate costs and validate requests before sending.

### Content Size Validation

```bash
# Check token count for a prompt before sending
echo "Analyze this large codebase and provide detailed recommendations" | gemapi count-tokens

# Sample output:
# Input: 12 tokens
# Estimated input cost: $0.000015 (with gemini-1.5-flash-latest)

# Check context size before making request
grove-context generate  # Generate context files
cat .grove/context .grove/cached-context | gemapi count-tokens

# Sample output:
# Input: 45,127 tokens  
# Estimated input cost: $0.056409 (with gemini-1.5-flash-latest)
# ⚠️  Large context detected - consider using caching
```

### Model Selection Based on Content

```bash
# For simple tasks, estimate with flash model
echo "Fix the typos in this documentation" | gemapi count-tokens -m gemini-1.5-flash-latest

# For complex analysis, estimate with pro model
echo "Perform comprehensive security audit of authentication system" | gemapi count-tokens -m gemini-1.5-pro-latest

# Compare costs between models
for model in gemini-1.5-flash-latest gemini-1.5-pro-latest; do
  echo "Complex analysis task" | gemapi count-tokens -m $model
done
```

### Batch Processing Estimation

```bash
# Estimate costs for processing multiple files
find docs/ -name "*.md" -exec sh -c '
  echo "=== Processing: $1 ==="
  echo "Summarize this document" | cat - "$1" | gemapi count-tokens
' _ {} \;

# Estimate total cost for batch operation
total_tokens=0
for file in docs/*.md; do
  tokens=$(echo "Process this file" | cat - "$file" | gemapi count-tokens | grep -o '[0-9,]* tokens' | tr -d ',')
  total_tokens=$((total_tokens + tokens))
done
echo "Total estimated tokens: $total_tokens"
```

## Advanced Integration Patterns

### Shell Script Integration

```bash
#!/bin/bash
# automated-review.sh - Automated code review script

set -e

# Setup
BRANCH=$(git branch --show-current)
REVIEW_DIR="reviews/$(date +%Y%m%d)"
mkdir -p "$REVIEW_DIR"

# Pre-flight check
echo "Estimating review cost..."
TOKENS=$(grove-context generate && cat .grove/context .grove/cached-context | gemapi count-tokens | grep -o '[0-9,]* tokens')
echo "Estimated tokens: $TOKENS"

# Generate review
echo "🔍 Performing automated code review for branch: $BRANCH"
gemapi request -f review-prompt.md -o "$REVIEW_DIR/review-$BRANCH.md"

# Check for errors
if [ $? -eq 0 ]; then
  echo "✅ Review completed: $REVIEW_DIR/review-$BRANCH.md"
  
  # Log usage
  gemapi query local --hours 1 --limit 1 >> "$REVIEW_DIR/usage-log.txt"
else
  echo "❌ Review failed"
  gemapi query local --errors --hours 1
  exit 1
fi
```

### Git Hook Integration

```bash
#!/bin/bash
# .git/hooks/pre-push - Automated documentation updates

echo "🤖 Updating documentation before push..."

# Check if documentation needs updating
if git diff --name-only HEAD~1 | grep -E '\.(go|ts|py)$' > /dev/null; then
  echo "Code changes detected, updating docs..."
  
  # Update API documentation
  gemapi request "Update the API documentation based on recent code changes" \
    -o docs/api-reference.md
  
  # Update changelog
  RECENT_COMMITS=$(git log --oneline HEAD~5..HEAD)
  echo "Generate changelog entries for these commits: $RECENT_COMMITS" | \
    gemapi request -o CHANGELOG-update.md
    
  echo "✅ Documentation updated"
else
  echo "No code changes detected, skipping doc update"
fi
```

### Continuous Integration

```bash
# .github/workflows/ai-review.yml snippet
- name: AI Code Review
  run: |
    # Setup gemapi
    echo "${{ secrets.GEMINI_API_KEY }}" | base64 -d > /tmp/key
    export GEMINI_API_KEY=$(cat /tmp/key)
    
    # Pre-flight cost check
    ESTIMATED_COST=$(grove-context generate && cat .grove/context | gemapi count-tokens | grep 'Estimated.*cost' | grep -o '\$[0-9.]*')
    echo "Estimated review cost: $ESTIMATED_COST"
    
    # Perform review if cost is reasonable
    if [[ $(echo "$ESTIMATED_COST < 1.00" | bc) -eq 1 ]]; then
      gemapi request "Review this pull request for code quality, security, and best practices" > ai-review.md
      gh pr comment --body-file ai-review.md
    else
      echo "Review cost too high ($ESTIMATED_COST), skipping AI review"
    fi
```

These usage patterns demonstrate how `gemapi` can be integrated into various development workflows, from simple one-off queries to complex automated processes. The key is to leverage caching for repeated analysis, monitor costs proactively, and use the right model for each task type.