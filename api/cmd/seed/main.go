package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/shule360/api/internal/config"
	"github.com/shule360/api/pkg/supabase"
)

// seedUser represents a demo staff user to create.
type seedUser struct {
	Email    string
	Password string
	FullName string
	Role     string
}

func main() {
	_ = godotenv.Load(".env")

	ctx := context.Background()

	// Load config (reuse existing config package)
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to Supabase
	sb, err := supabase.NewClient(ctx, cfg.DatabaseURL, cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer sb.Close()
	slog.Info("database connected")

	// Demo users to seed
	users := []seedUser{
		{Email: "principal@juakali.sch.ke", Password: "password123", FullName: "John Kamau", Role: "principal"},
		{Email: "bursar@juakali.sch.ke", Password: "password123", FullName: "Jane Wanjiku", Role: "bursar"},
		{Email: "teacher1@juakali.sch.ke", Password: "password123", FullName: "Peter Otieno", Role: "teacher"},
		{Email: "teacher2@juakali.sch.ke", Password: "password123", FullName: "Mary Akinyi", Role: "teacher"},
		{Email: "transport@juakali.sch.ke", Password: "password123", FullName: "David Mwangi", Role: "transport_manager"},
	}

	// Ensure super_admin staff record exists
	var superAdminID string
	err = sb.Pool.QueryRow(ctx,
		"SELECT id FROM staff WHERE email = $1 AND tenant_id = 'a0000000-0000-0000-0000-000000000001'",
		"admin@juakali.sch.ke",
	).Scan(&superAdminID)
	if err == nil {
		// exists
	} else {
		// Create super_admin staff record
		err = sb.Pool.QueryRow(ctx,
			"INSERT INTO staff (tenant_id, full_name, email, role, is_active) VALUES ($1, $2, $3, $4, true) RETURNING id",
			"a0000000-0000-0000-0000-000000000001", "Super Admin", "admin@juakali.sch.ke", "super_admin",
		).Scan(&superAdminID)
		if err != nil {
			slog.Error("failed to create super_admin staff record", "error", err)
		} else {
			slog.Info("created super_admin staff record", "id", superAdminID)
		}
	}

	// Add super_admin to the users list
	users = append(users, seedUser{Email: "admin@juakali.sch.ke", Password: "password123", FullName: "Super Admin", Role: "super_admin"})

	for _, u := range users {
		slog.Info("seeding user", "email", u.Email)

		// Step 1: Create Supabase Auth user
		userID, err := sb.CreateUser(ctx, u.Email, u.Password)
		if err != nil {
			// Check if user already exists
			existingID, getErr := findSupabaseUserID(ctx, sb, u.Email)
			if getErr != nil {
				slog.Error("failed to create or find supabase user", "email", u.Email, "error", err)
				continue
			}
			userID = existingID
			slog.Info("user already exists in supabase auth", "email", u.Email, "id", userID)
		} else {
			slog.Info("created supabase auth user", "email", u.Email, "id", userID)
		}

		// Step 2: Update staff record with supabase_user_id
		_, err = sb.Pool.Exec(ctx,
			"UPDATE staff SET supabase_user_id = $1::uuid WHERE email = $2 AND tenant_id = 'a0000000-0000-0000-0000-000000000001'",
			userID, u.Email,
		)
		if err != nil {
			slog.Error("failed to update staff record", "email", u.Email, "error", err)
			continue
		}
		slog.Info("updated staff record with supabase_user_id", "email", u.Email, "id", userID)
	}

	slog.Info("seed complete. Login credentials:")
	for _, u := range users {
		fmt.Printf("  %s / %s (%s)\n", u.Email, u.Password, u.Role)
	}
}

// findSupabaseUserID looks up a Supabase Auth user ID by email.
func findSupabaseUserID(ctx context.Context, sb *supabase.Client, email string) (string, error) {
	url := sb.URL() + "/auth/v1/admin/users?email=" + email
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", sb.ServiceKey())
	req.Header.Set("Authorization", "Bearer "+sb.ServiceKey())

	resp, err := sb.HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("lookup user: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lookup error (status %d): %s", resp.StatusCode, string(body))
	}

	var users []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &users); err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", fmt.Errorf("no user found")
	}
	return users[0].ID, nil
}

func ioReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
