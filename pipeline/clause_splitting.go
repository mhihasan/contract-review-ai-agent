package pipeline

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrClauseParse = errors.New("clause extraction: could not parse JSON after retries")

func parseClauses(raw string) ([]string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	var clauses []string
	if err := json.Unmarshal([]byte(s), &clauses); err != nil {
		return nil, err
	}
	if len(clauses) == 0 {
		return nil, errors.New("model returned empty clause array")
	}
	return clauses, nil
}
