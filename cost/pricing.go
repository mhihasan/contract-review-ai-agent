package cost

import "log"

var rates = map[string]struct{ InPerM, OutPerM float64 }{
	"openai:gpt-4o":                       {2.50, 10.00},
	"openai:gpt-4o-mini":                  {0.15, 0.60},
	"anthropic:claude-sonnet-4-6":         {3.00, 15.00},
	"anthropic:claude-haiku-4-5-20251001": {0.80, 4.00},
	"anthropic:claude-opus-4-8":           {15.00, 75.00},
}

func Estimate(provider, model string, in, out int) float64 {
	key := provider + ":" + model
	r, ok := rates[key]
	if !ok {
		log.Printf("cost: unknown model %q — treating cost as 0 (update cost/pricing.go)", key)
		return 0
	}
	return float64(in)/1_000_000*r.InPerM + float64(out)/1_000_000*r.OutPerM
}
