package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type config struct {
	Port      string `env:"PORT"`
	JWTSecret string `env:"JWT_SECRET"`
	Roles     roles  `env:"ROLES"`
}

type roles []string

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

func (r *roles) SetValue(s string) error {
	if s == "" {
		return fmt.Errorf("field value can't be empty")
	}

	roles := strings.Split(s, " ")
	for i := 0; i < len(roles); i++ {
		*r = append(*r, roles[i])
	}

	return nil
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

	err = os.Setenv("ROLES", "admin owner member")
	if err != nil {
		return fmt.Errorf("Error setting roles, err = %v", err)
	}

	return nil
}
