// Command platformuser creates the first platform administrator.
//
// Platform and group users are normally added from the web app (Platform →
// Users), but that screen needs a platform administrator to open it. This
// creates that first one, or any later one, from the command line:
//
//	go run ./cmd/platformuser -email ops@example.com -name "Jane Doe"
//
// It reads the same environment (.env) as the server, creates the sign-in
// account in Supabase Auth, and prints a generated password once.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/shule360/api/internal/config"
	"github.com/shule360/api/internal/platform"
	"github.com/shule360/api/pkg/supabase"
)

func main() {
	email := flag.String("email", "", "email address the administrator signs in with")
	name := flag.String("name", "", "the administrator's full name")
	flag.Parse()
	if *email == "" || *name == "" {
		flag.Usage()
		os.Exit(2)
	}

	_ = godotenv.Load(".env")
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	sb, err := supabase.NewClient(ctx, cfg.DatabaseURL, cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer sb.Close()

	user, err := platform.NewService(sb.Pool, sb).CreateUser(ctx, platform.UserInput{
		Email: *email, FullName: *name, Scope: "platform", Role: "super_admin",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create the administrator:", err)
		os.Exit(1)
	}

	fmt.Printf("Platform administrator created: %s <%s>\n", user.FullName, user.Email)
	if user.TempPassword != "" {
		fmt.Printf("Password (shown once, pass it on securely): %s\n", user.TempPassword)
	}
	if user.Note != "" {
		fmt.Println(user.Note)
	}
}
