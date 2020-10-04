package main

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type config struct {
	Port         string         `yaml:"port"`
	JWTSecret    string         `yaml:"jwt_secret"`
	Database     databaseConfig `yaml:"database"`
	EmailService emailService   `yaml:"email_service"`
}

type databaseConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	SSLMode  string `yaml:"ssl_mode"`
}

type emailService struct {
	EmailKey                    string `yaml:"email_key"`
	AccountValidationTemplateID string `yaml:"account_validation_template_id"`
}

func main() {
	cfg, err := ParseConfigFiles("./db_config.yaml", "./email_config.yaml", "./general_config.yaml")
	if err != nil {
		log.Printf("Error parsing config files: %v", err)
		return
	}

	log.Println("Parsed Configuration")
	log.Println(*cfg)

	return
}

func ParseConfigFiles(files ...string) (*config, error) {
	var cfg config

	for i := 0; i < len(files); i++ {
		err := cleanenv.ReadConfig(files[i], &cfg)
		if err != nil {
			log.Printf("Error reading configuration from file:%v", files[i])
			return nil, err
		}
	}

	return &cfg, nil
}
