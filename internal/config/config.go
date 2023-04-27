package config

import (
	"fmt"
	"os"
	"reflect"
	"sync"
)

var cfg *Config
var once sync.Once

// Config is the configuration for the application
type Config struct {
	Server
	PostgreSQL
	Process
}

type Process struct {
	Interval string `env:"PROCESS_INTERVAL" envDefault:"10"`
}

// Server is the configuration for the server
type Server struct {
	Port string `env:"PORT" envDefault:"8080"`
}

// Addr returns the address for the server
func (s Server) Addr() string {
	return fmt.Sprintf("%s:%s", "0.0.0.0", s.Port)
}

// PostgreSQL is the configuration for the database
type PostgreSQL struct {
	Driver          string `env:"DB_DRIVER" envDefault:"postgres"`
	Host            string `env:"DB_HOST" envDefault:"localhost"`
	Port            string `env:"DB_PORT" envDefault:"5432"`
	Database        string `env:"DB_DATABASE" envDefault:"enlabs_test_service"`
	Username        string `env:"DB_USERNAME" envDefault:"enlabs_test_service"`
	Password        string `env:"DB_PASSWORD" envDefault:"enlabs_test_service"`
	SSLMode         string `env:"DB_SSLMODE" envDefault:"disable"`
	MaxConnAttempts string `env:"DB_MAX_CONN_ATTEMPTS" envDefault:"5"`
}

// DSN returns the DSN for the database
func (c PostgreSQL) DSN() string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=%s",
		c.Driver,
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

// Load loads the configuration from environment variables
func Load() *Config {
	once.Do(func() {
		cfg = &Config{}
		cfgType := reflect.TypeOf(*cfg)
		cfgValue := reflect.ValueOf(cfg).Elem()

		for i := 0; i < cfgType.NumField(); i++ {
			field := cfgType.Field(i)
			fieldValue := cfgValue.Field(i)
			for j := 0; j < field.Type.NumField(); j++ {
				subField := field.Type.Field(j)
				envVar := subField.Tag.Get("env")
				envDefault := subField.Tag.Get("envDefault")
				value := getEnv(envVar, envDefault)

				fieldValue.Field(j).SetString(value)
			}
		}
	})

	return cfg
}

// getEnv retrieves the value of the environment variable named by the key or returns the defaultValue if not set
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}
