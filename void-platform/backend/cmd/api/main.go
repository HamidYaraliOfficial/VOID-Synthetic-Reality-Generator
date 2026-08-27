// Command api runs the VOID API Server: REST + WebSocket control plane for
// Universes, Scenarios, Entities, Snapshots, Export and the Scheduler.
package main

import (
	"log"
	"os"

	"void-platform/backend/internal/api"
	"void-platform/backend/internal/storage"
)

func main() {
	addr := getEnv("VOID_API_ADDR", ":8080")
	secret := getEnv("VOID_JWT_SECRET", "")
	if secret == "" {
		log.Println("WARNING: VOID_JWT_SECRET not set — using an insecure development default. Set it in production.")
		secret = "void-dev-secret-change-me"
	}
	if err := storage.EnsureDir("data/snapshots"); err != nil {
		log.Fatalf("failed to prepare data directory: %v", err)
	}
	log.Printf("VOID — Synthetic Reality Generator API starting on %s", addr)
	if err := api.Serve(addr, secret); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
