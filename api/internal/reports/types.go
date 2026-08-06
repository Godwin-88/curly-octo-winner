package reports

import (
	"time"

	"github.com/google/uuid"
)

// ReportCard is a CBC-compliant report card for a learner per term/year.
type ReportCard struct {
	ID                    uuid.UUID         `json:"id"`
	TenantID              uuid.UUID         `json:"tenant_id"`
	LearnerID             uuid.UUID         `json:"learner_id"`
	LearnerName           string            `json:"learner_name,omitempty"`
	Grade                 string            `json:"grade,omitempty"`
	Stream                string            `json:"stream,omitempty"`
	UPI                   string            `json:"upi,omitempty"`
	Term                  int               `json:"term"`
	Year                  int               `json:"year"`
	Status                string            `json:"status"`
	OverallRating         *int              `json:"overall_rating,omitempty"`
	CoreCompetencyRemarks map[string]string `json:"core_competency_remarks,omitempty"`
	TeacherComments       map[string]string `json:"teacher_comments,omitempty"`
	AttendanceSummary     map[string]any    `json:"attendance_summary,omitempty"`
	GeneratedBy           *uuid.UUID        `json:"generated_by,omitempty"`
	GeneratedAt           time.Time         `json:"generated_at"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	// Items populated on detail generation
	Items []ReportCardItem `json:"items,omitempty"`
}

// ReportCardItem is a line item on a report card (per sub-strand).
type ReportCardItem struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ReportCardID   uuid.UUID  `json:"report_card_id"`
	LearningAreaID *uuid.UUID `json:"learning_area_id,omitempty"`
	StrandID       *uuid.UUID `json:"strand_id,omitempty"`
	SubStrandID    *uuid.UUID `json:"sub_strand_id,omitempty"`
	LearningArea   string     `json:"learning_area,omitempty"`
	StrandName     string     `json:"strand_name,omitempty"`
	SubStrandName  string     `json:"sub_strand_name,omitempty"`
	RubricLevel    *int       `json:"rubric_level,omitempty"`
	RubricLabel    string     `json:"rubric_label,omitempty"`
	Comment        *string    `json:"comment,omitempty"`
	SortOrder      int        `json:"sort_order"`
	CreatedAt      time.Time  `json:"created_at"`
}

// GenerateReportCardRequest is the payload to generate a report card.
type GenerateReportCardRequest struct {
	Status                *string           `json:"status,omitempty"`
	OverallRating         *int              `json:"overall_rating,omitempty"`
	CoreCompetencyRemarks map[string]string `json:"core_competency_remarks,omitempty"`
	TeacherComments       map[string]string `json:"teacher_comments,omitempty"`
	GeneratedBy           *uuid.UUID        `json:"generated_by,omitempty"`
}

// UpdateReportCardRequest is a partial update payload.
type UpdateReportCardRequest struct {
	Status                *string           `json:"status,omitempty"`
	OverallRating         *int              `json:"overall_rating,omitempty"`
	CoreCompetencyRemarks map[string]string `json:"core_competency_remarks,omitempty"`
	TeacherComments       map[string]string `json:"teacher_comments,omitempty"`
}

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
