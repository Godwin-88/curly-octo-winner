package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-for-scope-tests"

// fakeSchools maps a school to its group (nil = no group).
type fakeSchools map[uuid.UUID]*uuid.UUID

func (f fakeSchools) SchoolGroup(_ context.Context, id uuid.UUID) (*uuid.UUID, bool, error) {
	group, found := f[id]
	return group, found, nil
}

// operatorStaff is the staff row the fake hands every operator.
var operatorStaff = uuid.New()

func (f fakeSchools) OperatorStaff(_ context.Context, _, operatorID uuid.UUID) (uuid.UUID, string, bool, error) {
	if operatorID == deactivatedOperator {
		return uuid.Nil, "", false, nil
	}
	// The role held now, which may differ from the one in the token.
	return operatorStaff, "bursar", true, nil
}

var deactivatedOperator = uuid.New()

func signed(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

type seen struct {
	Tenant       string `json:"tenant"`
	Scope        string `json:"scope"`
	Role         string `json:"role"`
	HeaderTenant string `json:"header_tenant"`
	HeaderStaff  string `json:"header_staff"`
}

// call runs a request through Auth + TenantRequired and reports what the
// handler would have seen.
func call(t *testing.T, schools SchoolLookup, token string, headers map[string]string) (int, seen, string) {
	t.Helper()
	handler := Auth(testSecret, false, schools)(TenantRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, _ := GetTenantID(r)
		role, _ := GetStaffRole(r)
		_ = json.NewEncoder(w).Encode(seen{
			Tenant: tenant.String(), Scope: GetSession(r).Scope, Role: role,
			HeaderTenant: r.Header.Get("X-Tenant-ID"), HeaderStaff: r.Header.Get("X-Staff-ID"),
		})
	})))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var got seen
	var failure struct {
		Code string `json:"code"`
	}
	body := rec.Body.Bytes()
	_ = json.Unmarshal(body, &got)
	_ = json.Unmarshal(body, &failure)
	return rec.Code, got, failure.Code
}

func TestSessionScopes(t *testing.T) {
	groupA, groupB := uuid.New(), uuid.New()
	schoolA, schoolB, loneSchool := uuid.New(), uuid.New(), uuid.New()
	schools := fakeSchools{schoolA: &groupA, schoolB: &groupB, loneSchool: nil}
	staffID, operatorID := uuid.New(), uuid.New()

	staffToken := signed(t, jwt.MapClaims{"tenant_id": schoolA.String(), "staff_id": staffID.String(), "role": "principal"})
	groupToken := signed(t, jwt.MapClaims{"operator_id": operatorID.String(), "scope": "group", "group_id": groupA.String(), "role": "principal"})
	platformToken := signed(t, jwt.MapClaims{"operator_id": operatorID.String(), "scope": "platform", "role": "super_admin"})

	t.Run("school staff are pinned to their own school", func(t *testing.T) {
		status, got, _ := call(t, schools, staffToken, nil)
		if status != http.StatusOK || got.Tenant != schoolA.String() || got.Scope != ScopeSchool {
			t.Fatalf("status %d, %+v", status, got)
		}
	})

	t.Run("school staff cannot widen themselves with the header", func(t *testing.T) {
		status, got, _ := call(t, schools, staffToken, map[string]string{
			SchoolHeader:  schoolB.String(),
			"X-Tenant-ID": schoolB.String(),
			"X-Staff-ID":  uuid.NewString(),
		})
		if status != http.StatusOK {
			t.Fatalf("status %d", status)
		}
		if got.Tenant != schoolA.String() || got.HeaderTenant != schoolA.String() || got.HeaderStaff != staffID.String() {
			t.Fatalf("a client-supplied school or identity leaked through: %+v", got)
		}
	})

	t.Run("a group user opens a school of their group", func(t *testing.T) {
		status, got, _ := call(t, schools, groupToken, map[string]string{SchoolHeader: schoolA.String()})
		if status != http.StatusOK || got.Tenant != schoolA.String() || got.Scope != ScopeGroup {
			t.Fatalf("status %d, %+v", status, got)
		}
		// Inside the school they act as their own staff row, with the role they
		// hold now rather than the one their token was issued with.
		if got.HeaderStaff != operatorStaff.String() || got.Role != "bursar" {
			t.Fatalf("operator staff/role = %s/%s, want %s/bursar", got.HeaderStaff, got.Role, operatorStaff)
		}
	})

	t.Run("a deactivated operator is refused at once, whatever their token says", func(t *testing.T) {
		token := signed(t, jwt.MapClaims{"operator_id": deactivatedOperator.String(), "scope": "platform", "role": "super_admin"})
		if status, _, _ := call(t, schools, token, map[string]string{SchoolHeader: schoolA.String()}); status != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", status)
		}
	})

	t.Run("a group user cannot open another group's school", func(t *testing.T) {
		for _, other := range []uuid.UUID{schoolB, loneSchool} {
			status, _, code := call(t, schools, groupToken, map[string]string{SchoolHeader: other.String()})
			if status != http.StatusNotFound || code != "SCHOOL_NOT_FOUND" {
				t.Errorf("school %s: status %d code %q, want 404 SCHOOL_NOT_FOUND", other, status, code)
			}
		}
	})

	t.Run("outside reach looks the same as missing", func(t *testing.T) {
		outside, _, codeOutside := call(t, schools, groupToken, map[string]string{SchoolHeader: schoolB.String()})
		missing, _, codeMissing := call(t, schools, groupToken, map[string]string{SchoolHeader: uuid.NewString()})
		if outside != missing || codeOutside != codeMissing {
			t.Fatalf("outside = %d %s, missing = %d %s", outside, codeOutside, missing, codeMissing)
		}
	})

	t.Run("a platform user opens any school that exists", func(t *testing.T) {
		for _, school := range []uuid.UUID{schoolA, schoolB, loneSchool} {
			status, got, _ := call(t, schools, platformToken, map[string]string{SchoolHeader: school.String()})
			if status != http.StatusOK || got.Tenant != school.String() {
				t.Errorf("school %s: status %d, %+v", school, status, got)
			}
		}
		status, _, code := call(t, schools, platformToken, map[string]string{SchoolHeader: uuid.NewString()})
		if status != http.StatusNotFound || code != "SCHOOL_NOT_FOUND" {
			t.Errorf("unknown school: status %d code %q, want 404", status, code)
		}
	})

	t.Run("no school chosen is refused with SCHOOL_REQUIRED", func(t *testing.T) {
		for name, token := range map[string]string{"group": groupToken, "platform": platformToken} {
			status, _, code := call(t, schools, token, nil)
			if status != http.StatusBadRequest || code != "SCHOOL_REQUIRED" {
				t.Errorf("%s: status %d code %q, want 400 SCHOOL_REQUIRED", name, status, code)
			}
		}
	})

	t.Run("an operator cannot supply a school through the identity headers", func(t *testing.T) {
		status, _, code := call(t, schools, platformToken, map[string]string{"X-Tenant-ID": schoolA.String()})
		if status != http.StatusBadRequest || code != "SCHOOL_REQUIRED" {
			t.Fatalf("status %d code %q, want 400 SCHOOL_REQUIRED", status, code)
		}
	})

	t.Run("a malformed school id is refused", func(t *testing.T) {
		status, _, code := call(t, schools, platformToken, map[string]string{SchoolHeader: "not-a-uuid"})
		if status != http.StatusBadRequest || code != "INVALID_SCHOOL" {
			t.Fatalf("status %d code %q, want 400 INVALID_SCHOOL", status, code)
		}
	})

	t.Run("a group token without a group, or an unknown scope, is refused", func(t *testing.T) {
		noGroup := signed(t, jwt.MapClaims{"operator_id": operatorID.String(), "scope": "group", "role": "principal"})
		if status, _, _ := call(t, schools, noGroup, map[string]string{SchoolHeader: schoolA.String()}); status != http.StatusUnauthorized {
			t.Errorf("group token without group_id: status %d, want 401", status)
		}
		odd := signed(t, jwt.MapClaims{"operator_id": operatorID.String(), "scope": "galaxy", "role": "super_admin", "tenant_id": schoolA.String()})
		if status, _, _ := call(t, schools, odd, nil); status != http.StatusUnauthorized {
			t.Errorf("unknown scope: status %d, want 401", status)
		}
	})

	t.Run("without a school lookup an operator can open nothing", func(t *testing.T) {
		status, _, _ := call(t, nil, platformToken, map[string]string{SchoolHeader: schoolA.String()})
		if status != http.StatusNotFound {
			t.Fatalf("status %d, want 404", status)
		}
	})
}

type fakeModules map[string]bool

func (f fakeModules) ModuleEnabled(_ context.Context, _ uuid.UUID, module string) (bool, error) {
	return f[module], nil
}

// A module the school does not have is refused before the handler runs; one
// it has passes through.
func TestRequireModule(t *testing.T) {
	school := uuid.New()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusOK) })

	for _, tc := range []struct {
		name    string
		modules fakeModules
		want    int
		reaches bool
	}{
		{"enabled", fakeModules{"finance": true}, http.StatusOK, true},
		{"not enabled", fakeModules{"communications": true}, http.StatusForbidden, false},
	} {
		called = false
		req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyTenantID, school))
		rec := httptest.NewRecorder()
		RequireModule(tc.modules, "finance", "Finance")(next).ServeHTTP(rec, req)
		if rec.Code != tc.want || called != tc.reaches {
			t.Errorf("%s: status %d reached=%v, want %d reached=%v", tc.name, rec.Code, called, tc.want, tc.reaches)
		}
	}
}
