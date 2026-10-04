package reports

import (
	"time"

	"github.com/google/uuid"
)

// --- Analytics ---

// SchoolOverview is the top-level dashboard summary.
type SchoolOverview struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	LearnerCount int64     `json:"learner_count"`
}

// AlertLearner is an at-risk learner flagged for multi-strand underperformance.
type AlertLearner struct {
	LearnerID        uuid.UUID `json:"learner_id"`
	LearnerName      string    `json:"learner_name"`
	Grade            string    `json:"grade"`
	Stream           string    `json:"stream"`
	Term             int       `json:"term"`
	Year             int       `json:"year"`
	OverallAvgRubric float64   `json:"overall_avg_rubric"`
	AttendanceRate   float64   `json:"attendance_rate"`
	AssessedAreas    int64     `json:"assessed_areas"`
}

// StrandCoverage is a heatmap row of strand coverage per class.
type StrandCoverage struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	Grade              string    `json:"grade"`
	Stream             string    `json:"stream"`
	LearningAreaID     uuid.UUID `json:"learning_area_id"`
	LearningArea       string    `json:"learning_area"`
	StrandID           uuid.UUID `json:"strand_id"`
	StrandName         string    `json:"strand_name"`
	Term               int       `json:"term"`
	Year               int       `json:"year"`
	SubStrandsAssessed int64     `json:"sub_strands_assessed"`
	LearnersAssessed   int64     `json:"learners_assessed"`
}

// CompetencyDistribution is the learner count per rubric level per strand.
type CompetencyDistribution struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Grade        string    `json:"grade"`
	Stream       string    `json:"stream"`
	StrandID     uuid.UUID `json:"strand_id"`
	StrandName   string    `json:"strand_name"`
	Term         int       `json:"term"`
	Year         int       `json:"year"`
	RubricLevel  int       `json:"rubric_level"`
	LearnerCount int64     `json:"learner_count"`
}

// TeacherVelocity is the assessment count per teacher per week.
type TeacherVelocity struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	TeacherID       uuid.UUID `json:"teacher_id"`
	TeacherName     string    `json:"teacher_name"`
	Term            int       `json:"term"`
	Year            int       `json:"year"`
	WeekStart       time.Time `json:"week_start"`
	AssessmentCount int64     `json:"assessment_count"`
}

// LearnerPortfolio is per-learner aggregates joined with attendance.
type LearnerPortfolio struct {
	TenantID              uuid.UUID `json:"tenant_id"`
	LearnerID             uuid.UUID `json:"learner_id"`
	LearnerName           string    `json:"learner_name"`
	Grade                 string    `json:"grade"`
	Stream                string    `json:"stream"`
	Term                  int       `json:"term"`
	Year                  int       `json:"year"`
	LearningAreasAssessed int64     `json:"learning_areas_assessed"`
	OverallAvgRubric      float64   `json:"overall_avg_rubric"`
	AttendanceRate        float64   `json:"attendance_rate"`
}

// LearningAreaPerformance is per-learner performance per learning area.
type LearningAreaPerformance struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	LearnerID       uuid.UUID `json:"learner_id"`
	Term            int       `json:"term"`
	Year            int       `json:"year"`
	LearningAreaID  uuid.UUID `json:"learning_area_id"`
	LearningArea    string    `json:"learning_area"`
	AssessmentCount int64     `json:"assessment_count"`
	AvgRubricLevel  float64   `json:"avg_rubric_level"`
}
