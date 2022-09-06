package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is an application configuration structure
type Config struct {
	Database struct {
		Host        string `yaml:"host" env:"DB_HOST" env-description:"Database host"`
		Port        string `yaml:"port" env:"DB_PORT" env-description:"Database port"`
		Username    string `yaml:"username" env:"DB_USER" env-description:"Database user name"`
		Password    string `env:"DB_PASSWORD" env-description:"Database user password"`
		Name        string `yaml:"db-name" env:"DB_NAME" env-description:"Database name"`
		Connections int    `yaml:"connections" env:"DB_CONNECTIONS" env-description:"Total number of database connections"`
	} `yaml:"database"`
	Server struct {
		Host string `yaml:"host" env:"SRV_HOST,HOST" env-description:"Server host" env-default:"localhost"`
		Port string `yaml:"port" env:"SRV_PORT,PORT" env-description:"Server port" env-default:"8080"`
	} `yaml:"server"`
	Greeting string `env:"GREETING" env-description:"Greeting phrase" env-default:"Hello!"`
}

// ConnectDB connects to an abstract database
func ConnectDB(host, port, user, password, name string, conn int) (*sql.DB, error) {
	db, err := sql.Open("some database",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, name))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(conn)
	return db, nil
}

func main() {
	// comment this out and try setting envvars yourself
	setupEnv()

	var greeterCfg Config
	var specialCfg Config
	var wg = &sync.WaitGroup{}

	// start the greeter service with the first set of configuration
	// connect to the DB (example)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// read configuration from the non-namespaced environment variables
		if err := cleanenv.ReadEnv(&greeterCfg); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		ConnectDB(greeterCfg.Database.Host, greeterCfg.Database.Port, greeterCfg.Database.Username,
			greeterCfg.Database.Password, greeterCfg.Database.Name, greeterCfg.Database.Connections)
		server := createServer(greeterCfg.Server.Host, greeterCfg.Server.Port, greeterCfg.Greeting)
		server.ListenAndServe()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// read the other configuration from environment variables,
		// using namespace "SPECIAL_"
		if err := cleanenv.ReadEnv(&specialCfg, "SPECIAL_"); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		// connect to the other DB (example)
		ConnectDB(specialCfg.Database.Host, specialCfg.Database.Port, specialCfg.Database.Username,
			specialCfg.Database.Password, specialCfg.Database.Name, specialCfg.Database.Connections)

		server := createServer(specialCfg.Server.Host, specialCfg.Server.Port, specialCfg.Greeting)
		server.ListenAndServe()
	}()

	wg.Wait()
}

func createServer(host string, port string, message string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", message)
	})
	server := http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: mux,
	}

	return &server
}

func setupEnv() {
	os.Setenv("DB_HOST", "db-host-1")
	os.Setenv("DB_PORT", "4567")
	os.Setenv("DB_USER", "mickey")
	os.Setenv("DB_PASSWORD", "notgonnaputithere")
	os.Setenv("DB_NAME", "greeter_db")
	os.Setenv("DB_CONNECTIONS", "2000")
	os.Setenv("SRV_HOST", "localhost")
	os.Setenv("SRV_PORT", "8080")
	os.Setenv("GREETING", "Greetings to you!")

	os.Setenv("SPECIAL_DB_HOST", "db-host-2")
	os.Setenv("SPECIAL_DB_PORT", "4567")
	os.Setenv("SPECIAL_DB_USER", "minnie")
	os.Setenv("SPECIAL_DB_PASSWORD", "neitherhere")
	os.Setenv("SPECIAL_DB_NAME", "special_db")
	os.Setenv("SPECIAL_DB_CONNECTIONS", "2000")
	os.Setenv("SPECIAL_SRV_HOST", "localhost")
	os.Setenv("SPECIAL_SRV_PORT", "8081")
	os.Setenv("SPECIAL_GREETING", "Very Special Greetings to you!")
}
