package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type config struct {
	Port      string        `env:"PORT"`
	JWTSecret string        `env:"JWT_SECRET"`
	DB        url.URL       `env:"DB"`
	Start     time.Time     `env:"START"`
	TTL       time.Duration `env:"TTL" env-required:"true"`
}

func main() {
	err := setEnvValues()
	if err != nil {
		panic(err)
	}

	var cfg config
	err = cleanenv.ReadEnv(&cfg)
	if err != nil {
		panic(err)
	}

	log.Println("Parsed Configuration")
	log.Println(cfg)
	return
}

func setEnvValues() error {
	err := os.Setenv("PORT", "8080")
	if err != nil {
		return fmt.Errorf("Error setting port, err = %v", err)
	}

	err = os.Setenv("JWT_SECRET", "random_secret")
	if err != nil {
		return fmt.Errorf("Error setting jwt secret, err = %v", err)
	}

	err = os.Setenv("START", time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("Error setting start, err = %v", err)
	}

	err = os.Setenv("TTL", "17s")
	if err != nil {
		return fmt.Errorf("Error setting ttl, err = %v", err)
	}

	err = os.Setenv("DB", "redis://user:password@redishost:1234")
	if err != nil {
		return fmt.Errorf("Error setting URL, err = %v", err)
	}

	return nil
}
