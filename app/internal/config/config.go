package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDev   bool `env:"IS_DEV" env-default:"false"`
	IsDebug bool `env:"IS_DEBUG" env-default:"false"`

	OpenAI struct {
		APIKey          string `env:"OPENAI_API_KEY" env-required:"true"`
		BaseURL         string `env:"OPENAI_BASE_URL" env-default:"https://api.openai.com/v1"`
		RateLimitPerMin int    `env:"RATE_LIMIT_PER_MIN" env-default:"60"`
	}
	HTTP struct {
		Port int `env:"PORT" env-default:"8080"`
	}
	Repository struct {
		Type      string `env:"REPOSITORY_TYPE" env-default:"memory"`
		SQLiteDSN string `env:"SQLITE_DSN" env-default:"sessions.db"`
	}
}

// Singleton: Config should only ever be created once.
var instance *Config

// Once is an object that will perform exactly one action.
var once sync.Once

// GetConfig returns pointer to Config.
func GetConfig() *Config {
	// Calls the function if and only if Do is being called for the first time for this instance of Once
	once.Do(func() {
		log.Print("collecting config...")

		// Config initialization
		instance = &Config{}

		// Read environment variables into the instance of the Config
		if err := cleanenv.ReadEnv(instance); err != nil {
			// If something is wrong
			helpText := "Environment variables error:"
			// Returns a description of environment variables with a custom header - helpText
			help, err := cleanenv.GetDescription(instance, &helpText)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(help)
			log.Printf("%+v\n", instance)

			log.Fatal(err)
		}
	})
	return instance
}
