// Package models provides centralized model definitions for Google Gemini models.
package models

// Model represents an LLM model with its metadata.
type Model struct {
	ID       string  // Full API model ID (e.g., "gemini-2.5-pro")
	Alias    string  // Short alias, empty if none (Gemini IDs are already short)
	Provider string  // Provider name (e.g., "Google")
	Note     string  // Human-readable description
	Input    float64 // Input price per million tokens (short context)
	Output   float64 // Output price per million tokens
	Legacy   bool    // Whether this is a legacy model
}

// DefaultModel is the recommended default model to use.
const DefaultModel = "gemini-2.5-pro"

// LongContextThreshold is the token count above which long context pricing applies.
// Gemini uses 200k for pricing tiers (Pro and 3 Pro have different rates above 200k).
const LongContextThreshold int32 = 200_000

// Models returns all available Google Gemini models.
// Pricing as of Feb 2026 - see https://ai.google.dev/gemini-api/docs/pricing
func Models() []Model {
	return []Model{
		// Gemini 3.1 models (preview)
		{
			ID:       "gemini-3.1-pro-preview",
			Alias:    "",
			Provider: "Google",
			Note:     "Latest intelligent multimodal and agentic model",
			Input:    2.00,  // $2.00 <=200k, $4.00 >200k
			Output:   12.00, // $12.00 <=200k, $18.00 >200k
			Legacy:   false,
		},
		// Gemini 3 models (preview)
		{
			ID:       "gemini-3-pro-preview",
			Alias:    "",
			Provider: "Google",
			Note:     "Most intelligent multimodal and agentic model",
			Input:    2.00,  // $2.00 <=200k, $4.00 >200k
			Output:   12.00, // $12.00 <=200k, $18.00 >200k
			Legacy:   false,
		},
		{
			ID:       "gemini-3-flash-preview",
			Alias:    "",
			Provider: "Google",
			Note:     "Fastest intelligent model with search/grounding",
			Input:    0.50,
			Output:   3.00,
			Legacy:   false,
		},
		// Gemini 2.5 models (current stable)
		{
			ID:       "gemini-2.5-pro",
			Alias:    "",
			Provider: "Google",
			Note:     "Advanced thinking model for complex problems",
			Input:    1.25,  // $1.25 <=200k, $2.50 >200k
			Output:   10.00, // $10.00 <=200k, $15.00 >200k
			Legacy:   false,
		},
		{
			ID:       "gemini-2.5-flash",
			Alias:    "",
			Provider: "Google",
			Note:     "Best price-performance, large scale processing",
			Input:    0.30,
			Output:   2.50,
			Legacy:   false,
		},
		{
			ID:       "gemini-2.5-flash-lite",
			Alias:    "",
			Provider: "Google",
			Note:     "Ultra-fast, cost-efficient, high throughput",
			Input:    0.10,
			Output:   0.40,
			Legacy:   false,
		},
		// Gemini 2.0 models (legacy)
		{
			ID:       "gemini-2.0-flash",
			Alias:    "",
			Provider: "Google",
			Note:     "Second gen workhorse model (legacy)",
			Input:    0.10,
			Output:   0.40,
			Legacy:   true,
		},
		{
			ID:       "gemini-2.0-flash-lite",
			Alias:    "",
			Provider: "Google",
			Note:     "Second gen fast model (legacy)",
			Input:    0.075,
			Output:   0.30,
			Legacy:   true,
		},
	}
}

// Aliases returns a map of alias -> full model ID for all models with aliases.
// Note: Gemini model IDs are already short, so this is mostly empty.
func Aliases() map[string]string {
	aliases := make(map[string]string)
	for _, m := range Models() {
		if m.Alias != "" {
			aliases[m.Alias] = m.ID
		}
	}
	return aliases
}

// ResolveAlias expands a model alias to its full API ID, or returns the input unchanged.
func ResolveAlias(model string) string {
	if fullID, ok := Aliases()[model]; ok {
		return fullID
	}
	return model
}

// CurrentModels returns only non-legacy models (for TUI pickers, etc.).
func CurrentModels() []Model {
	var current []Model
	for _, m := range Models() {
		if !m.Legacy {
			current = append(current, m)
		}
	}
	return current
}

// GetPricing returns input and output price per million tokens for a model.
// Returns default Pro pricing if model not found.
func GetPricing(model string) (input, output float64) {
	// Resolve alias first
	model = ResolveAlias(model)

	for _, m := range Models() {
		if m.ID == model {
			return m.Input, m.Output
		}
	}
	// Default to Pro pricing
	return 1.25, 10.00
}
