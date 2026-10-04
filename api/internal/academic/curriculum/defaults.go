package curriculum

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shule360/api/pkg/apperr"
)

// The seven core competencies and eight values of the Competency Based
// Curriculum are the same for every school and every grade, so a new school
// starts with them.
var defaultCompetencies = []string{
	"Communication and Collaboration",
	"Critical Thinking and Problem Solving",
	"Creativity and Imagination",
	"Citizenship",
	"Digital Literacy",
	"Learning to Learn",
	"Self-Efficacy",
}

var defaultValues = []string{
	"Love", "Responsibility", "Respect", "Unity", "Peace", "Patriotism", "Social Justice", "Integrity",
}

// Learning areas by level. These are a starting list a school edits: it adds
// the strands and sub-strands it teaches from its KICD curriculum designs.
var (
	prePrimaryAreas = []string{
		"Language Activities", "Mathematical Activities", "Creative Activities",
		"Environmental Activities", "Religious Activities",
	}
	lowerPrimaryAreas = []string{
		"Indigenous Language Activities", "Kiswahili Language Activities", "English Language Activities",
		"Mathematical Activities", "Religious Education Activities", "Environmental Activities", "Creative Activities",
	}
	upperPrimaryAreas = []string{
		"English", "Kiswahili", "Mathematics", "Religious Education", "Science and Technology",
		"Agriculture and Nutrition", "Social Studies", "Creative Arts",
	}
	juniorSchoolAreas = []string{
		"English", "Kiswahili", "Mathematics", "Religious Education", "Social Studies", "Integrated Science",
		"Pre-Technical Studies", "Agriculture and Nutrition", "Creative Arts and Sports",
	}
)

// DefaultLearningAreas returns the usual learning areas for a grade written
// the way the school writes its classes ("PP1", "Grade 4"), or nil for a
// grade there is no list for.
func DefaultLearningAreas(grade string) []string {
	switch strings.ToLower(strings.Join(strings.Fields(grade), " ")) {
	case "pp1", "pp2", "pp 1", "pp 2":
		return prePrimaryAreas
	case "grade 1", "grade 2", "grade 3":
		return lowerPrimaryAreas
	case "grade 4", "grade 5", "grade 6":
		return upperPrimaryAreas
	case "grade 7", "grade 8", "grade 9":
		return juniorSchoolAreas
	}
	return nil
}

// SeedDefaults gives a new school the core competencies and values. It adds
// only what is missing, so it is safe to run for a school that has some.
func SeedDefaults(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	for table, names := range map[string][]string{"core_competencies": defaultCompetencies, "values": defaultValues} {
		// table is one of the two literals above, never request input.
		q := fmt.Sprintf(`
			INSERT INTO %[1]s (tenant_id, name)
			SELECT $1, n FROM unnest($2::text[]) AS n
			WHERE NOT EXISTS (SELECT 1 FROM %[1]s x WHERE x.tenant_id = $1 AND lower(x.name) = lower(n))`, table)
		if _, err := tx.Exec(ctx, q, tenantID, names); err != nil {
			return fmt.Errorf("seed %s: %w", table, err)
		}
	}
	return nil
}

// AddDefaultLearningAreas adds the usual learning areas for a grade that the
// school does not have yet, and returns how many were added.
func (s *Service) AddDefaultLearningAreas(ctx context.Context, tenantID uuid.UUID, grade string) (int, error) {
	grade = strings.Join(strings.Fields(grade), " ")
	if grade == "" {
		return 0, apperr.Invalid("Choose the grade to add learning areas for.")
	}
	names := DefaultLearningAreas(grade)
	if names == nil {
		return 0, apperr.Invalid("There is no standard list of learning areas for %q. Add them one by one.", grade)
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO learning_areas (tenant_id, name, grade_level)
		SELECT $1, n, $2::text FROM unnest($3::text[]) AS n
		WHERE NOT EXISTS (
			SELECT 1 FROM learning_areas x
			WHERE x.tenant_id = $1 AND lower(x.grade_level) = lower($2::text) AND lower(x.name) = lower(n))`,
		tenantID, grade, names)
	if err != nil {
		return 0, fmt.Errorf("add learning areas: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
