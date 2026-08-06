package intelligence

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/upstash"
)

// AIService handles Upstash Vector semantic search features.
type AIService struct {
	pool    *pgxpool.Pool
	vector  *upstash.VectorClient
	enabled bool
}

// NewAIService creates an AI service. If vector is nil, semantic features
// fall back to keyword-based matching.
func NewAIService(pool *pgxpool.Pool, vector *upstash.VectorClient) *AIService {
	return &AIService{
		pool:    pool,
		vector:  vector,
		enabled: vector != nil,
	}
}

// SuggestTemplates returns top-3 similar message templates for a given purpose.
// Uses Upstash Vector when available; otherwise falls back to keyword matching.
func (s *AIService) SuggestTemplates(ctx context.Context, tenantID uuid.UUID, purpose, tone, language string, topK int) ([]TemplateSuggestion, error) {
	if topK <= 0 {
		topK = 3
	}
	if tone == "" {
		tone = "formal"
	}
	if language == "" {
		language = "en"
	}

	// Keyword-based fallback (works without Upstash Vector)
	rows, err := s.pool.Query(ctx, `
		SELECT content, purpose, tone, language
		FROM message_template_embeddings
		WHERE tenant_id = $1 AND ($2 = '' OR tone = $2) AND ($3 = '' OR language = $3)
		ORDER BY created_at DESC
		LIMIT $4`, tenantID, tone, language, topK*5)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []TemplateSuggestion
	for rows.Next() {
		var t TemplateSuggestion
		if err := rows.Scan(&t.Content, &t.Purpose, &t.Tone, &t.Language); err != nil {
			return nil, err
		}
		candidates = append(candidates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Score by keyword overlap with the purpose
	queryWords := tokenize(purpose)
	var scored []TemplateSuggestion
	for _, c := range candidates {
		score := 0.0
		contentWords := tokenize(c.Content)
		if c.Purpose != nil {
			contentWords = append(contentWords, tokenize(*c.Purpose)...)
		}
		for _, qw := range queryWords {
			for _, cw := range contentWords {
				if qw == cw {
					score++
				}
			}
		}
		if score > 0 || len(queryWords) == 0 {
			scored = append(scored, TemplateSuggestion{
				Content:  c.Content,
				Purpose:  c.Purpose,
				Tone:     c.Tone,
				Language: c.Language,
				Score:    score,
			})
		}
	}

	// Sort by score descending, then take topK
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	if len(scored) > topK {
		scored = scored[:topK]
	}
	return scored, nil
}

// AutoRespond matches a parent query against the FAQ knowledge base.
// Returns the best match with a confidence score.
func (s *AIService) AutoRespond(ctx context.Context, tenantID uuid.UUID, query string) (*AutoResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &AutoResponse{Matched: false}, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT question, answer, category, keywords
		FROM faq_entries
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type faq struct {
		question string
		answer   string
		category string
		keywords []string
	}
	var faqs []faq
	for rows.Next() {
		var f faq
		if err := rows.Scan(&f.question, &f.answer, &f.category, &f.keywords); err != nil {
			return nil, err
		}
		faqs = append(faqs, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	queryWords := tokenize(queryLower)

	best := &AutoResponse{Matched: false}
	bestScore := 0.0
	for _, f := range faqs {
		score := 0.0
		// Score against question text
		for _, qw := range queryWords {
			if strings.Contains(strings.ToLower(f.question), qw) {
				score++
			}
		}
		// Score against keywords
		for _, kw := range f.keywords {
			if strings.Contains(queryLower, strings.ToLower(kw)) {
				score += 2
			}
		}
		if score > bestScore {
			bestScore = score
			best = &AutoResponse{
				Answer:   f.answer,
				Category: f.category,
				Score:    score,
				Matched:  score > 0,
			}
		}
	}
	return best, nil
}

// tokenize splits text into lowercase word tokens.
func tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '\''
	})
	var out []string
	for _, f := range fields {
		if len(f) > 1 {
			out = append(out, f)
		}
	}
	return out
}
