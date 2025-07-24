package config

import (
	"log"
	"os"
	"github.com/joho/godotenv"
)

func LoadEnv() {
	env := os.Getenv("ENV")            // k8s or local
	role := os.Getenv("ROLE")          // producer or consumer

	// Default to local dev if not set
	if env == "" {
		env = "local"
	}
	if role == "" {
		role = "producer"
	}

	envFile := "/app/.env." + role + "." + env // Full path to mounted file

	err := godotenv.Load(envFile)
	if err != nil {
		log.Printf("⚠️  Warning: could not load env file: %s\n", envFile)
	} else {
		log.Printf("✅ Loaded env file: %s\n", envFile)
	}
}
