package comms

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/pkg/pgxutil"
)

// Variables a message may contain, and what each is filled in with.
//
// {{fee_balance}} is deliberately absent: until the balance shown to a parent
// is reconciled against payments made straight to the paybill, texting it
// would state a figure the school cannot stand behind.
var supportedVariables = map[string]string{
	"parent_name":  "the recipient's name",
	"learner_name": "the learner(s) of that parent",
	"class":        "the class of those learner(s)",
	"school_name":  "the school's name",
}

// SupportedVariables lists the {{names}} a message may use, sorted.
func SupportedVariables() []string {
	names := make([]string, 0, len(supportedVariables))
	for name := range supportedVariables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type learnerInfo struct {
	names   string
	classes string
}

// personalise fills in the message for every recipient and records the longest
// result, which is what the length limit and the estimate are based on.
//
// A variable that cannot be filled in for someone refuses the whole send. The
// alternative — sending "Dear {{parent_name}}" or a blank — is worse than not
// sending, and the school is told exactly who is affected.
func (s *CommsService) personalise(ctx context.Context, tenantID uuid.UUID, content string, out *preparedAudience) error {
	used := sms.Variables(content)
	if len(used) == 0 {
		out.segments = sms.CountSegments(content)
		return nil
	}
	for _, name := range used {
		if _, ok := supportedVariables[name]; !ok {
			return invalid("{{%s}} is not something that can be filled in. Available: {{%s}}.",
				name, strings.Join(SupportedVariables(), "}}, {{"))
		}
	}

	uses := func(name string) bool {
		for _, u := range used {
			if u == name {
				return true
			}
		}
		return false
	}

	var schoolName string
	if uses("school_name") {
		if err := s.pool.QueryRow(ctx, `SELECT name FROM tenants WHERE id = $1`, tenantID).Scan(&schoolName); err != nil {
			return fmt.Errorf("load school name: %w", err)
		}
	}

	learners := map[uuid.UUID]learnerInfo{}
	if uses("learner_name") || uses("class") {
		var guardianIDs []uuid.UUID
		for _, r := range out.valid {
			if r.Type == "guardian" && r.ID != uuid.Nil {
				guardianIDs = append(guardianIDs, r.ID)
			}
		}
		if len(guardianIDs) > 0 {
			rows, err := s.pool.Query(ctx, `
				SELECT gid,
				       string_agg(DISTINCT l.full_name, ', ' ORDER BY l.full_name),
				       string_agg(DISTINCT trim(l.grade || ' ' || COALESCE(l.stream, '')), ', '
				                  ORDER BY trim(l.grade || ' ' || COALESCE(l.stream, '')))
				FROM learners l
				CROSS JOIN LATERAL unnest(l.guardian_ids) AS gid
				WHERE l.tenant_id = $1 AND l.is_active = true AND gid = ANY($2::uuid[])
				GROUP BY gid
			`, tenantID, pgxutil.UUIDArray(guardianIDs))
			if err != nil {
				return fmt.Errorf("load learners for personalisation: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var id uuid.UUID
				var info learnerInfo
				if err := rows.Scan(&id, &info.names, &info.classes); err != nil {
					return fmt.Errorf("scan learners for personalisation: %w", err)
				}
				learners[id] = info
			}
			if err := rows.Err(); err != nil {
				return err
			}
		}
	}

	missingCount := map[string]int{}
	missingExample := map[string]string{}
	for i := range out.valid {
		r := &out.valid[i]
		data := map[string]string{}
		if name := strings.TrimSpace(r.Name); name != "" {
			data["parent_name"] = name
		}
		if schoolName != "" {
			data["school_name"] = schoolName
		}
		if info, ok := learners[r.ID]; ok && r.Type == "guardian" {
			if info.names != "" {
				data["learner_name"] = info.names
			}
			if info.classes != "" {
				data["class"] = info.classes
			}
		}
		for _, name := range used {
			if _, ok := data[name]; !ok {
				missingCount[name]++
				if missingExample[name] == "" {
					missingExample[name] = firstNonEmpty(strings.TrimSpace(r.Name), r.Phone)
				}
			}
		}

		rendered := sms.InjectVariables(content, data)
		r.rendered = &rendered
		if seg := sms.CountSegments(rendered); seg.Units > out.segments.Units ||
			(seg.Units == out.segments.Units && seg.Length > out.segments.Length) {
			out.segments = seg
		}
	}

	if len(out.valid) == 0 {
		out.segments = sms.CountSegments(content)
	}

	for _, name := range used {
		if n := missingCount[name]; n > 0 {
			return invalid("{{%s}} cannot be filled in for %d of %d recipients (for example %s). Remove it from the message, or choose an audience where everyone has %s.",
				name, n, len(out.valid), missingExample[name], supportedVariables[name])
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
