# Best Practices

This guide covers recommended best practices for using `gemapi` effectively, securely, and economically. Following these practices will help you optimize costs, maintain security, and integrate `gemapi` smoothly into your development workflow.

## API Key Security

Protecting your Gemini API key is crucial for security and cost control.

### Recommended Approaches

**Do:**
- Use environment variables for API keys:
  ```bash
  export GEMINI_API_KEY="your-api-key-here"
  ```

- Use secure command execution in `grove.yml`:
  ```yaml
  gemini:
    api_key_command: "gcloud auth print-access-token"
  ```

- Use credential management tools:
  ```yaml
  gemini:
    api_key_command: "aws ssm get-parameter --name /gemini/api-key --with-decryption --query Parameter.Value --output text"
  ```

- Store keys in CI/CD secret management:
  ```yaml
  # GitHub Actions example
  env:
    GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
  ```

**Don't:**
- Store plaintext API keys in `grove.yml`:
  ```yaml
  # ❌ AVOID THIS
  gemini:
    api_key: "AIzaSyD..."  # Never commit actual keys
  ```

- Commit API keys to version control
- Share API keys in chat, email, or documentation
- Use the same API key across multiple environments

### Key Rotation

```bash
# Example rotation workflow
# 1. Generate new key in Google Cloud Console
# 2. Update environment variable
export GEMINI_API_KEY="new-api-key"

# 3. Test with a simple request
gemapi request "test message"

# 4. Update any stored configurations
# 5. Revoke old key in Google Cloud Console
```

## Cost Management

Controlling costs is essential when working with large codebases and frequent API calls.

### Cost-Conscious Model Selection

**Do:**
- Use `gemini-1.5-flash-latest` for simple tasks:
  ```bash
  # Text summarization, simple Q&A, basic code review
  gemapi request -m gemini-1.5-flash-latest "Summarize this README file"
  
  # Token counting and estimation
  gemapi count-tokens -m gemini-1.5-flash-latest "Simple prompt"
  ```

- Use `gemini-1.5-pro-latest` for complex analysis:
  ```bash
  # Architecture analysis, security reviews, complex reasoning
  gemapi request -m gemini-1.5-pro-latest "Perform comprehensive security audit"
  ```

- Pre-estimate costs before expensive operations:
  ```bash
  # Check cost before large requests
  grove-context generate
  cat .grove/context .grove/cached-context | gemapi count-tokens -m gemini-1.5-pro-latest
  ```

**Don't:**
- Use expensive models for simple tasks
- Make large requests without cost estimation
- Ignore model pricing differences

### Caching Strategy

**Do:**
- Cache large, stable codebases:
  ```
  @enable-cache
  **/*.go
  **/*.py
  **/*.ts
  !**/*_test.go
  !**/node_modules/**
  ```

- Use cache directives strategically:
  ```
  @enable-cache
  @expire-time 7d        # Cache for a week
  **/*.go
  ```

- Monitor cache effectiveness:
  ```bash
  # Check cache usage and savings
  gemapi query tokens --hours 168
  gemapi cache list
  ```

**Don't:**
- Enable caching without understanding costs
- Cache rapidly changing files
- Forget to monitor cache usage

### Regular Cost Monitoring

**Do:**
- Check local usage daily:
  ```bash
  # Quick daily check
  gemapi query local --hours 24
  ```

- Review weekly costs:
  ```bash
  # Comprehensive weekly review
  gemapi query billing --days 7 \
    --dataset-id billing_export \
    --table-id gcp_billing_export_v1_ABCDEF_123456
  ```

- Set up automated cost alerts:
  ```bash
  #!/bin/bash
  # cost-alert.sh - Run daily via cron
  
  DAILY_COST=$(gemapi query billing --days 1 | grep "Total.*Cost" | grep -o '\$[0-9.]*')
  THRESHOLD=10.00
  
  if (( $(echo "$DAILY_COST > $THRESHOLD" | bc -l) )); then
    echo "⚠️ Daily Gemini costs exceeded threshold: $DAILY_COST > $THRESHOLD"
    # Send alert via email, Slack, etc.
  fi
  ```

**Don't:**
- Ignore cost monitoring until bills arrive
- Run expensive queries without budgets
- Let cache costs accumulate unchecked

## Effective Context Management

Optimizing context inclusion improves both performance and costs.

### Writing Effective .grove/rules

**Do:**
- Include only relevant files:
  ```
  # Good: Specific and targeted
  src/**/*.ts
  docs/**/*.md
  **/*.config.js
  !**/dist/**
  !**/node_modules/**
  !**/*_test.ts
  ```

- Use exclusion patterns liberally:
  ```
  # Exclude large, irrelevant directories
  !**/vendor/**
  !**/node_modules/**
  !**/build/**
  !**/dist/**
  !**/.git/**
  !**/*.log
  !**/*.tmp
  ```

- Structure rules from general to specific:
  ```
  # Start with broad inclusion
  **/*.go
  **/*.mod
  
  # Add specific exclusions
  !**/*_test.go
  !**/testdata/**
  !**/vendor/**
  ```

**Don't:**
- Include test files in production analysis
- Include build artifacts or generated files
- Use overly broad patterns like `**/*`

### Context Regeneration vs. Recaching

**Do:**
- Use `--regenerate` after changing `.grove/rules`:
  ```bash
  # After modifying rules file
  gemapi request --regenerate "Analyze updated file selection"
  ```

- Use `--recache` after significant code changes:
  ```bash
  # After major refactoring or new features
  gemapi request --recache "Review the updated architecture"
  ```

- Understand the difference:
  - `--regenerate`: Updates which files are included based on rules
  - `--recache`: Updates cached content of existing files

**Don't:**
- Use `--regenerate` when only file contents changed
- Use `--recache` when you meant to change file selection
- Forget to regenerate after rules changes

## Workflow Tips

Integrating `gemapi` effectively into development workflows.

### Output Management

**Do:**
- Use output files for documentation:
  ```bash
  # Generate reusable documentation
  gemapi request "Create API documentation" -o docs/api-reference.md
  
  # Save analysis results
  gemapi request "Security audit report" -o security-audit-$(date +%Y%m%d).md
  ```

- Pipe output for further processing:
  ```bash
  # Extract specific information
  gemapi request "List all API endpoints" | grep -E '^(GET|POST|PUT|DELETE)'
  
  # Convert formats
  gemapi request -f analysis-prompt.md | pandoc -o report.pdf
  ```

- Structure output for different audiences:
  ```bash
  # Technical detailed analysis
  gemapi request -m gemini-1.5-pro-latest -f technical-review.md -o detailed-analysis.md
  
  # Executive summary
  gemapi request -m gemini-1.5-flash-latest "Summarize the technical analysis in business terms" -o executive-summary.md
  ```

**Don't:**
- Let important analysis output scroll off the screen
- Lose track of expensive analysis results
- Mix different types of analysis in single files

### Shell Script Integration

**Do:**
- Build reusable scripts:
  ```bash
  #!/bin/bash
  # review-script.sh
  
  set -e
  
  BRANCH=${1:-$(git branch --show-current)}
  OUTPUT_DIR="reviews/$(date +%Y%m%d)"
  mkdir -p "$OUTPUT_DIR"
  
  echo "🔍 Reviewing branch: $BRANCH"
  
  # Pre-flight cost check
  TOKENS=$(grove-context generate && cat .grove/context | gemapi count-tokens | grep -o '[0-9,]* tokens' | tr -d ',')
  if [ "$TOKENS" -gt 50000 ]; then
    echo "⚠️ Large context detected ($TOKENS tokens). Continue? (y/N)"
    read -r response
    [ "$response" = "y" ] || exit 1
  fi
  
  # Perform analysis
  gemapi request "Review this code for quality, security, and performance" \
    -o "$OUTPUT_DIR/review-$BRANCH.md"
  
  echo "✅ Review completed: $OUTPUT_DIR/review-$BRANCH.md"
  ```

- Use error handling:
  ```bash
  # Check for errors and retry
  if ! gemapi request "Analysis prompt" -o result.md; then
    echo "❌ Request failed, checking recent errors..."
    gemapi query local --errors --hours 1
    
    # Retry with simpler prompt or different model
    gemapi request -m gemini-1.5-flash-latest "Simplified analysis" -o result.md
  fi
  ```

**Don't:**
- Write scripts without error handling
- Ignore cost checks in automated scripts
- Hard-code sensitive information in scripts

### Automation Best Practices

**Do:**
- Set reasonable timeouts and limits:
  ```bash
  # Add timeouts to prevent hanging
  timeout 300 gemapi request "Complex analysis" || echo "Request timed out"
  ```

- Log automation usage:
  ```bash
  # Track automated usage
  echo "$(date): Automated review completed" >> automation.log
  gemapi query local --limit 1 >> automation.log
  ```

- Use different models for different automation levels:
  ```bash
  # Quick automated checks
  gemapi request -m gemini-1.5-flash-latest "Basic code quality check"
  
  # Detailed scheduled reviews
  gemapi request -m gemini-1.5-pro-latest "Comprehensive security audit"
  ```

**Don't:**
- Run expensive automation without monitoring
- Use automation for tasks requiring human judgment
- Automate without testing cost implications first

### Development Environment Integration

**Do:**
- Set up environment-specific configurations:
  ```bash
  # Development environment
  export GEMINI_API_KEY="dev-key"
  echo "@enable-cache" > .grove/rules
  
  # Production analysis
  export GEMINI_API_KEY="prod-key"
  echo -e "@enable-cache\n@expire-time 24h" > .grove/rules
  ```

- Use project-specific settings:
  ```yaml
  # grove.yml for the project
  gemini:
    api_key_command: "gcloud auth print-access-token"
    default_model: "gemini-1.5-flash-latest"
  ```

**Don't:**
- Mix development and production API keys
- Use the same configuration across all projects
- Forget to configure API keys in new environments

Following these best practices will help you use `gemapi` effectively while maintaining security, controlling costs, and integrating smoothly with your development workflow. Regular monitoring and adjustment of these practices will ensure continued optimal usage as your projects evolve.