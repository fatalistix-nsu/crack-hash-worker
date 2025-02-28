package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"time"
)

func MustRead() *Config {
	config := Config{}

	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	if err := cleanenv.ReadEnv(&config); err != nil {
		panic(err)
	}

	return &config
}

type Config struct {
	Deployment DeploymentConfig
	Manager    ManagerConfig
	Worker     WorkerConfig
}

type DeploymentConfig struct {
	Port            int           `env:"DEPLOYMENT_PORT"`
	ShutdownTimeout time.Duration `env:"DEPLOYMENT_SHUTDOWN_TIMEOUT"`
}

type ManagerConfig struct {
	Address string `env:"MANAGER_ADDRESS"`
}

type WorkerConfig struct {
	GoroutineCount uint64 `env:"WORKER_GOROUTINE_COUNT"`
}
