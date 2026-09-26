package intelligence

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/groq"
	"github.com/shule360/api/pkg/upstash"
)

// AIService handles AI-powered features using Groq LLM + Upstash Vector fallback.
type AIService struct {
	pool   *pgxpool.Pool
	vector *upstash.VectorClient
	groq   *groq.Client
}

// NewAIService creates an AI service.
func NewAIService(pool *pgxpool.Pool, vector *upstash.VectorClient, groqClient *groq.Client) *AIService {
	return &AIService{
		pool:   pool,
		vector: vector,
		groq:   groqClient,
	}
}

// SuggestTemplates returns top-3 similar message templates for a given purpose.
// Uses Groq for intelligent generation when available, falls back to keyword matching.
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

	if s.groq != nil {
		systemPrompt := fmt.Sprintf(`You are an SMS communication assistant for a Kenyan school management system.
Generate up to %d SMS message templates for: %s.
Tone: %s. Language: %s.
Return only the message content, one per line, no numbering.`, topK, purpose, tone, language)
		resp, err := s.groq.Complete(ctx, systemPrompt, purpose)
		if err == nil {
			var out []TemplateSuggestion
			lines := strings.Split(resp, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				out = append(out, TemplateSuggestion{
					Content:  line,
					Purpose:  &purpose,
					Tone:     tone,
					Language: language,
					Score:    1.0,
				})
			}
			if len(out) > topK {
				out = out[:topK]
			}
			if len(out) > 0 {
				return out, nil
			}
		}
	}

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

// AutoRespond matches a parent query against the FAQ knowledge base or generates
// an intelligent response via Groq when no FAQ match is found.
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
		for _, qw := range queryWords {
			if strings.Contains(strings.ToLower(f.question), qw) {
				score++
			}
		}
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

	if best.Matched {
		return best, nil
	}

	if s.groq != nil {
		faqContext := "No matching FAQ found. Provide a helpful response based on general school knowledge."
		if len(faqs) > 0 {
			var parts []string
			for _, f := range faqs {
				parts = append(parts, fmt.Sprintf("Q: %s\nA: %s", f.question, f.answer))
			}
			faqContext = "Available FAQ entries:\n" + strings.Join(parts, "\n\n")
		}
		systemPrompt := fmt.Sprintf(`You are a helpful school communications assistant for a Kenyan K-12 school.
%s

If you cannot answer confidently, respond with: "I don't have that information right now. Please contact the school office for assistance."
Keep responses concise, friendly, and under 160 characters when possible.`, faqContext)
		resp, err := s.groq.Complete(ctx, systemPrompt, query)
		if err == nil {
			return &AutoResponse{
				Answer:   resp,
				Category: "ai_generated",
				Score:    0.5,
				Matched:  true,
			}, nil
		}
	}

	return best, nil
}

// PortfolioSummary generates a CBC portfolio summary using Groq when available,
// otherwise returns a basic summary from observation notes.
func (s *AIService) PortfolioSummary(ctx context.Context, tenantID uuid.UUID, learnerID uuid.UUID, term, year int) (*PortfolioSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT l.full_name, COUNT(a.id) AS note_count,
		       ROUND(AVG(a.rubric_level), 2) AS avg_rubric
		FROM learners l
		LEFT JOIN assessments a ON a.tenant_id = l.tenant_id AND a.learner_id = l.id
			AND a.term = $3 AND a.year = $4
		WHERE l.tenant_id = $1 AND l.id = $2
		GROUP BY l.id, l.full_name
	`, tenantID, learnerID, term, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var learnerName string
	var noteCount int64
	var avgRubric float64
	if rows.Next() {
		if err := rows.Scan(&learnerName, &noteCount, &avgRubric); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("%s has %d assessment observations this term with an average rubric level of %.2f.",
		learnerName, noteCount, avgRubric)

	if s.groq != nil && noteCount > 0 {
		assessRows, err := s.pool.Query(ctx, `
			SELECT la.name, str.name, a.rubric_level, a.note
			FROM assessments a
			JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
			JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
			JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
			WHERE a.tenant_id = $1 AND a.learner_id = $2 AND a.term = $3 AND a.year = $4
			ORDER BY la.name, str.name
		`, tenantID, learnerID, term, year)
		if err == nil {
			defer assessRows.Close()
			var observations []string
			for assessRows.Next() {
				var la, str, note string
				var level int
				if err := assessRows.Scan(&la, &str, &level, &note); err == nil {
					label := "Below Expectation"
					if level == 2 {
						label = "Approaching Expectation"
					} else if level == 3 {
						label = "Meeting Expectation"
					} else if level == 4 {
						label = "Exceeding Expectation"
					}
					obs := fmt.Sprintf("- %s / %s: %s", la, str, label)
					if note != "" {
						obs += fmt.Sprintf(" (%s)", note)
					}
					observations = append(observations, obs)
				}
			}
			if len(observations) > 0 {
				systemPrompt := `You are a CBC (Competency-Based Curriculum) portfolio summarizer for Kenyan schools.
Generate a concise, professional learner portfolio summary based on the observation notes below.
Write 2-3 sentences highlighting strengths and areas for growth.`
				resp, err := s.groq.Complete(ctx, systemPrompt, strings.Join(observations, "\n"))
				if err == nil {
					summary = resp
				}
			}
		}
	}

	return &PortfolioSummary{
		LearnerID:   learnerID,
		LearnerName: learnerName,
		Term:        term,
		Year:        year,
		Summary:     summary,
		NoteCount:   noteCount,
	}, nil
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
