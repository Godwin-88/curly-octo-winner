-- 021_analytics_views.sql
-- Analytics views for Reports & Analytics dashboards (EPIC 6)

-- Learning area performance per learner per term (for report card aggregation)
CREATE OR REPLACE VIEW learning_area_performance AS
SELECT
    a.tenant_id,
    a.learner_id,
    a.term,
    a.year,
    la.id AS learning_area_id,
    la.name AS learning_area,
    COUNT(a.id) AS assessment_count,
    ROUND(AVG(a.rubric_level)::numeric, 2) AS avg_rubric_level
FROM assessments a
JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
GROUP BY a.tenant_id, a.learner_id, a.term, a.year, la.id, la.name;

-- Strand coverage heatmap: which strands have been assessed per class
CREATE OR REPLACE VIEW strand_coverage AS
SELECT
    a.tenant_id,
    l.grade,
    l.stream,
    la.id AS learning_area_id,
    la.name AS learning_area,
    str.id AS strand_id,
    str.name AS strand_name,
    a.term,
    a.year,
    COUNT(DISTINCT s.id) AS sub_strands_assessed,
    COUNT(DISTINCT a.learner_id) AS learners_assessed
FROM assessments a
JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id
GROUP BY a.tenant_id, l.grade, l.stream, la.id, la.name, str.id, str.name, a.term, a.year;

-- Competency distribution: % of learners at each rubric level per strand
CREATE OR REPLACE VIEW competency_distribution AS
SELECT
    a.tenant_id,
    l.grade,
    l.stream,
    str.id AS strand_id,
    str.name AS strand_name,
    a.term,
    a.year,
    a.rubric_level,
    COUNT(DISTINCT a.learner_id) AS learner_count
FROM assessments a
JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
GROUP BY a.tenant_id, l.grade, l.stream, str.id, str.name, a.term, a.year, a.rubric_level;

-- Teacher assessment velocity: count of assessments recorded per teacher per period
CREATE OR REPLACE VIEW teacher_assessment_velocity AS
SELECT
    a.tenant_id,
    st.id AS teacher_id,
    st.full_name AS teacher_name,
    a.term,
    a.year,
    date_trunc('week', a.created_at) AS week_start,
    COUNT(a.id) AS assessment_count
FROM assessments a
JOIN staff st ON st.id = a.teacher_id AND st.tenant_id = a.tenant_id
GROUP BY a.tenant_id, st.id, st.full_name, a.term, a.year, date_trunc('week', a.created_at);

-- School overview: aggregate stats for the analytics dashboard
CREATE OR REPLACE VIEW school_overview AS
SELECT
    tenant_id,
    COUNT(*) AS learner_count
FROM learners
GROUP BY tenant_id;

-- Learner portfolio: aggregate rubric avg per learner per term joined with attendance
-- attendance rate uses term date ranges (Jan-Apr T1, May-Aug T2, Sep-Dec T3)
CREATE OR REPLACE VIEW learner_portfolio AS
SELECT
    lap.tenant_id,
    lap.learner_id,
    l.full_name AS learner_name,
    l.grade,
    l.stream,
    lap.term,
    lap.year,
    COUNT(DISTINCT lap.learning_area_id) AS learning_areas_assessed,
    ROUND(AVG(lap.avg_rubric_level)::numeric, 2) AS overall_avg_rubric,
    COALESCE(att.attendance_rate, 0) AS attendance_rate
FROM learning_area_performance lap
JOIN learners l ON l.id = lap.learner_id AND l.tenant_id = lap.tenant_id
LEFT JOIN (
    SELECT
        tenant_id, learner_id,
        CASE
            WHEN date_part('month', date) BETWEEN 1 AND 4 THEN 1
            WHEN date_part('month', date) BETWEEN 5 AND 8 THEN 2
            ELSE 3
        END AS term,
        date_part('year', date)::int AS year,
        ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'present') / NULLIF(COUNT(*), 0), 2) AS attendance_rate
    FROM attendance
    GROUP BY tenant_id, learner_id,
        CASE
            WHEN date_part('month', date) BETWEEN 1 AND 4 THEN 1
            WHEN date_part('month', date) BETWEEN 5 AND 8 THEN 2
            ELSE 3
        END,
        date_part('year', date)::int
) att ON att.learner_id = lap.learner_id AND att.tenant_id = lap.tenant_id AND att.term = lap.term AND att.year = lap.year
GROUP BY lap.tenant_id, lap.learner_id, l.full_name, l.grade, l.stream, lap.term, lap.year, att.attendance_rate;
