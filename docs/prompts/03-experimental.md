Generate a brief experimental features section for grove-gemini.

## Requirements
Create a concise warning section about experimental features, particularly caching:

1. **Opening**: Brief introduction that these features are experimental and may change

2. **Context Caching (⚠️ CRITICAL WARNING)**:
   - Include a CRITICAL WARNING similar to grove-context about substantial unexpected charges
   - Briefly explain the risk: misconfigured TTL or frequent cache changes = very high costs
   - State clearly: NOT recommended for general use
   - Mention the --cache flag and cache TTL settings are experimental

3. **Observability Integration**:
   - One paragraph about grove-hooks integration being experimental
   - Note that session tracking features are still evolving

Keep the entire section brief and focused on warnings rather than detailed documentation.