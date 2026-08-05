package nemis

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

// NEMISLearner represents basic learner information returned by the NEMIS API.
type NEMISLearner struct {
	UPI         string `json:"upi"`
	FullName    string `json:"full_name"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	Grade       string `json:"grade"`
	SchoolCode  string `json:"school_code"`
}

// NEMISSearchRequest is the request payload for learner search.
type NEMISSearchRequest struct {
	FullName    string `json:"full_name"`
	DateOfBirth string `json:"date_of_birth,omitempty"`
	SchoolCode  string `json:"school_code,omitempty"`
}

// NEMISClient is the interface for NEMIS (National Education Management
// Information System) integration. This is stubbed in development and
// swapped to live credentials in production without code changes.
type NEMISClient interface {
	// ValidateUPI checks if a UPI exists and returns basic learner info.
	ValidateUPI(ctx context.Context, upi string) (*NEMISLearner, error)

	// SearchLearner searches by name + DOB (for enrollment lookup).
	SearchLearner(ctx context.Context, req NEMISSearchRequest) ([]NEMISLearner, error)
}

// ErrNotFound is returned when a learner is not found in NEMIS.
var ErrNotFound = errors.New("nemis: learner not found")

// ErrInvalidUPI is returned when the UPI format is invalid.
var ErrInvalidUPI = errors.New("nemis: invalid UPI format")

// testUPIPattern matches NEMIS UPI format "TESTDDDDDDDD" (TEST + 8 digits).
var testUPIPattern = regexp.MustCompile(`^TEST\d{8}$`)

// realUPIPattern matches the standard Kenyan NEMIS UPI format (16 alphanumeric chars).
var realUPIPattern = regexp.MustCompile(`^[A-Z0-9]{16}$`)

// SandboxNEMISClient is the stub implementation used in development.
type SandboxNEMISClient struct{}

// NewSandboxNEMISClient creates a new sandbox NEMIS client.
func NewSandboxNEMISClient() *SandboxNEMISClient {
	return &SandboxNEMISClient{}
}

// ValidateUPI returns mock data for UPIs matching pattern "TEST\d{8}".
func (c *SandboxNEMISClient) ValidateUPI(ctx context.Context, upi string) (*NEMISLearner, error) {
	// Validate format
	if !testUPIPattern.MatchString(upi) && !realUPIPattern.MatchString(upi) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidUPI, upi)
	}

	// Only TEST-prefixed UPIs are recognized by the sandbox
	if !testUPIPattern.MatchString(upi) {
		return nil, ErrNotFound
	}

	// Derive deterministic mock data from the UPI
	last4 := upi[len(upi)-4:]
	digits := []rune(last4)

	gender := "F"
	if int(digits[0]-'0')%2 == 0 {
		gender = "M"
	}

	grade := "Grade 4"
	switch int(digits[0]-'0') % 3 {
	case 1:
		grade = "Grade 5"
	case 2:
		grade = "Grade 6"
	}

	// Validate the sandbox is being used
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &NEMISLearner{
		UPI:         upi,
		FullName:    "Sandbox Learner " + last4,
		DateOfBirth: fmt.Sprintf("201%d-0%d-15", 2+int(digits[0]-'0')%4, 1+int(digits[1]-'0')%9),
		Gender:      gender,
		Grade:       grade,
		SchoolCode:  "TEST-SCHOOL-001",
	}, nil
}

// SearchLearner searches by name + DOB in the sandbox (returns empty results).
func (c *SandboxNEMISClient) SearchLearner(ctx context.Context, req NEMISSearchRequest) ([]NEMISLearner, error) {
	// Sandbox always returns no results for searches
	return []NEMISLearner{}, nil
}

// RealNEMISClient is the production implementation skeleton.
// TODO: Implement with actual NEMIS REST API credentials.
type RealNEMISClient struct {
	baseURL string
	apiKey  string
	http    interface {
		Do(ctx context.Context, method, path string, payload any, out any) error
	}
}

// NewRealNEMISClient creates a real NEMIS client (requires production credentials).
func NewRealNEMISClient(baseURL, apiKey string) *RealNEMISClient {
	return &RealNEMISClient{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// ValidateUPI checks a UPI against the live NEMIS API.
func (c *RealNEMISClient) ValidateUPI(ctx context.Context, upi string) (*NEMISLearner, error) {
	// Implement with real NEMIS REST API when credentials are available
	return nil, errors.New("nemis: real client not yet implemented - use sandbox for development")
}

// SearchLearner searches the live NEMIS API.
func (c *RealNEMISClient) SearchLearner(ctx context.Context, req NEMISSearchRequest) ([]NEMISLearner, error) {
	// Implement with real NEMIS REST API when credentials are available
	return nil, errors.New("nemis: real client not yet implemented - use sandbox for development")
}
