# Observability Guide

`gemapi` provides comprehensive observability through the `gemapi query` command, which pulls data from multiple sources to give you complete visibility into your Gemini API usage, costs, and performance. This guide covers each query subcommand and how to use them effectively.

## query local

The `query local` command analyzes local JSONL logs stored on your machine, providing immediate insight into requests made from your development environment.

### What it does

- Queries locally stored request logs (JSONL format)
- Provides detailed information about each request including tokens, costs, latency, and git context
- Offers filtering and analysis capabilities for development workflow optimization

### Information Available

- **Request Details**: Timestamp, model used, prompt content preview
- **Token Usage**: Input tokens, output tokens, total tokens per request
- **Cost Analysis**: Estimated costs per request and cumulative costs
- **Performance Metrics**: Response latency and request success/failure status
- **Git Context**: Branch, commit hash, and repository state at request time
- **Cache Information**: Whether cache was used, cache hit/miss status
- **Error Details**: Full error messages and debugging information for failed requests

### Example Usage

```bash
# View requests from the last 24 hours (default)
gemapi query local

# View requests from the last 3 days
gemapi query local --hours 72

# Show only failed requests for debugging
gemapi query local --errors

# Filter by specific model
gemapi query local --model gemini-1.5-pro-latest

# Limit results and show requests from last week
gemapi query local --hours 168 --limit 50
```

### Sample Output

```
┌─────────────────────┬─────────────────┬─────────┬─────────┬──────────┬────────┬─────────────┐
│ Timestamp           │ Model           │ Input   │ Output  │ Cost     │ Time   │ Status      │
├─────────────────────┼─────────────────┼─────────┼─────────┼──────────┼────────┼─────────────┤
│ 2024-09-30 14:23:15 │ gemini-1.5-pro  │ 45,127  │ 1,234   │ $0.0847  │ 2.34s  │ ✅ Success  │
│ 2024-09-30 13:45:22 │ gemini-1.5-flash│ 12,456  │ 567     │ $0.0234  │ 1.12s  │ ✅ Success  │
│ 2024-09-30 12:30:44 │ gemini-1.5-pro  │ 34,567  │ 0       │ $0.0000  │ 0.45s  │ ❌ Error    │
└─────────────────────┴─────────────────┴─────────┴─────────┴──────────┴────────┴─────────────┘

Total Requests: 156
Success Rate: 94.2%
Total Tokens: 2,847,329 (input: 2,234,567, output: 612,762)
Total Cost: $12.47
Average Latency: 1.89s
```

## query metrics

The `query metrics` command fetches aggregated metrics from Google Cloud Monitoring, providing project-wide visibility into API usage patterns.

### What it does

- Queries Google Cloud Monitoring for Gemini API metrics
- Provides aggregated request counts, error rates, and performance statistics
- Offers project-wide visibility across all users and services

### Data Sources

- **Service**: Google Cloud Monitoring
- **Metrics**: Request counts, error rates, latency percentiles
- **Scope**: Project-wide metrics across all API consumers

### Information Available

- **Request Volume**: Total requests over time periods
- **Error Rates**: Success/failure ratios and error percentages
- **Performance Metrics**: Average latency, 95th percentile response times
- **Usage Patterns**: Hourly, daily, and weekly usage trends

### Example Usage

```bash
# Get metrics for the last 24 hours
gemapi query metrics --project-id your-project

# View metrics for the last 48 hours
gemapi query metrics --project-id your-project --hours 48

# Use default project from config
gemapi query metrics --hours 72
```

### Sample Output

```
Gemini API Metrics (Last 24 Hours)
Project: your-project-id

Request Volume:
├─ Total Requests: 1,247
├─ Successful Requests: 1,184 (94.9%)
├─ Failed Requests: 63 (5.1%)
└─ Requests/Hour: 52 avg (peak: 89 at 14:00)

Performance:
├─ Average Latency: 1.89s
├─ 95th Percentile: 3.42s
├─ 99th Percentile: 7.23s
└─ Fastest Response: 0.23s

Error Breakdown:
├─ Rate Limit Errors: 34 (54% of errors)
├─ Invalid Request: 18 (29% of errors)
├─ Service Unavailable: 8 (13% of errors)
└─ Other Errors: 3 (4% of errors)
```

## query tokens

The `query tokens` command provides detailed token usage information from Google Cloud Logging, offering insights into the actual token consumption patterns from the Gemini API.

### What it does

- Queries Google Cloud Logging for detailed token usage information
- Provides token-level details directly from the Gemini API logs
- Enables detailed analysis of token consumption patterns for cost optimization

### Data Sources

- **Service**: Google Cloud Logging
- **Logs**: Gemini API request/response logs
- **Scope**: Detailed token information from API perspective

### Information Available

- **Token Breakdown**: Prompt tokens vs completion tokens
- **Cache Information**: Cache hit/miss rates and token savings
- **Model-Specific Usage**: Token consumption by model type
- **Time-Series Data**: Token usage trends over time
- **Cost Estimation**: Detailed cost calculations based on actual token usage

### Example Usage

```bash
# Get token details for the last 24 hours
gemapi query tokens --project-id your-project

# Analyze token usage over the last week
gemapi query tokens --project-id your-project --hours 168

# Enable debug output for detailed analysis
gemapi query tokens --project-id your-project --debug
```

### Sample Output

```
Token Usage Analysis (Last 24 Hours)
Project: your-project-id

Token Summary:
├─ Total Tokens Processed: 2,847,329
├─ Prompt Tokens: 2,234,567 (78.5%)
├─ Completion Tokens: 612,762 (21.5%)
└─ Cache Hits: 1,823,445 tokens (64.1% cache hit rate)

Model Breakdown:
├─ gemini-1.5-pro-latest: 1,456,789 tokens (51.2%)
├─ gemini-1.5-flash: 1,234,567 tokens (43.4%)
└─ gemini-1.5-pro: 155,973 tokens (5.4%)

Cache Performance:
├─ Cached Tokens Served: 1,823,445
├─ Tokens Saved vs Direct: 1,456,234 (44.4% savings)
├─ Cache Hit Rate: 64.1%
└─ Estimated Cost Savings: $8.34

Hourly Distribution:
14:00 ████████████████████████████████████████ 234,567 tokens
13:00 ██████████████████████████████████ 189,234 tokens
12:00 ████████████████████████████ 156,789 tokens
11:00 ████████████████████████ 134,567 tokens
```

## query billing

The `query billing` command pulls cost data directly from BigQuery billing exports, providing the most accurate and detailed cost analysis available.

### What it does

- Queries BigQuery billing export tables for SKU-level cost information
- Provides detailed cost breakdown including Gemini API usage and caching costs
- Enables precise financial analysis and cost attribution

### Prerequisites

**Critical Requirement**: You must have a BigQuery billing export configured in your GCP account before using this command. This export automatically sends billing data to BigQuery tables for analysis.

#### Setting Up BigQuery Billing Export

1. **Navigate to Cloud Billing**: Go to the Google Cloud Console billing section
2. **Configure Export**: Set up billing export to BigQuery
3. **Choose Dataset**: Select or create a BigQuery dataset for billing data
4. **Table Creation**: Google automatically creates daily tables with billing data

### Data Sources

- **Service**: BigQuery billing export tables
- **Data**: Complete SKU-level cost information
- **Scope**: All GCP services including detailed Gemini API costs

### Information Available

- **SKU-Level Costs**: Detailed breakdown by service and SKU
- **Usage Amounts**: Actual usage quantities and units
- **Time-Based Analysis**: Daily, weekly, monthly cost trends
- **Service Attribution**: Costs attributed to specific GCP services
- **Currency Information**: Multi-currency support for global billing

### Example Usage

```bash
# Query billing data with required parameters
gemapi query billing \
  --project-id your-project \
  --dataset-id your_billing_dataset \
  --table-id your_billing_table

# Analyze costs over the last 7 days
gemapi query billing \
  --project-id your-project \
  --dataset-id billing_export \
  --table-id gcp_billing_export_v1_ABCDEF_123456 \
  --days 7

# Use default project from config
gemapi query billing \
  --dataset-id billing_export \
  --table-id gcp_billing_export_v1_ABCDEF_123456 \
  --days 30
```

### Required Flags

- `--dataset-id`: The BigQuery dataset containing your billing export
- `--table-id`: The specific billing table (usually follows pattern `gcp_billing_export_v1_{BILLING_ACCOUNT_ID}`)

### Sample Output

```
Billing Analysis (Last 7 Days)
Project: your-project-id
Dataset: billing_export.gcp_billing_export_v1_ABCDEF_123456

Gemini API Costs:
├─ Total Gemini Costs: $127.45
├─ Input Token Costs: $89.23 (70.0%)
├─ Output Token Costs: $31.12 (24.4%)
└─ Cache Storage Costs: $7.10 (5.6%)

SKU Breakdown:
├─ Gemini Pro Input Tokens: $67.89 (53.3%)
├─ Gemini Pro Output Tokens: $23.45 (18.4%)
├─ Gemini Flash Input Tokens: $21.34 (16.7%)
├─ Gemini Flash Output Tokens: $7.67 (6.0%)
└─ Cached Content Storage: $7.10 (5.6%)

Daily Trend:
2024-09-30: $24.56 ████████████████████████████████████████
2024-09-29: $18.23 ██████████████████████████████
2024-09-28: $22.11 ████████████████████████████████████
2024-09-27: $19.87 ██████████████████████████████████
2024-09-26: $16.45 ██████████████████████████
2024-09-25: $14.12 ████████████████████████
2024-09-24: $12.11 ████████████████████

Usage Statistics:
├─ Total Requests: 1,456
├─ Average Cost/Request: $0.0875
├─ Highest Daily Cost: $24.56 (2024-09-30)
└─ Cost Trend: ↗ +12.4% vs previous week
```

## Combining Observability Sources

Each query source provides different perspectives on your API usage:

### Development Workflow
```bash
# Quick local check during development
gemapi query local --hours 1

# Check if errors are affecting project
gemapi query metrics --hours 4

# Deep dive into token optimization
gemapi query tokens --hours 24

# Monthly cost review
gemapi query billing --days 30
```

### Troubleshooting Workflow
```bash
# 1. Check recent local errors
gemapi query local --errors --hours 2

# 2. Verify if it's a project-wide issue
gemapi query metrics --hours 2

# 3. Analyze token usage patterns
gemapi query tokens --hours 2 --debug

# 4. Check cost impact
gemapi query billing --days 1
```

### Optimization Workflow
```bash
# 1. Analyze usage patterns
gemapi query local --hours 168  # Last week

# 2. Check cache effectiveness
gemapi query tokens --hours 168

# 3. Review cost trends
gemapi query billing --days 30

# 4. Validate improvements
gemapi query metrics --hours 24
```

The observability features in `gemapi` provide comprehensive visibility into your Gemini API usage from multiple angles - local development perspective, project-wide monitoring, detailed token analysis, and precise cost tracking. Using these tools together gives you complete control over your API usage optimization and cost management.