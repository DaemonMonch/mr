package mr

import (
	"io"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v3"
)

type JobConfig struct {
	Jobs       map[string]*Job
	configFile string
	Output     io.Writer
}

type Job struct {
	Cmd     string
	Args    []string
	Envs    []string
	EnvFile string `yaml:"env_file"`
	Path    string
	Name    string

	HealthCheck *HealthCheck
}

type HealthCheck struct {
	Protocal       string
	Endpoint       string
	Address        string
	Port           int
	MaxAwaitTime   time.Duration
	CheckInterval  time.Duration
	CheckDelayTime time.Duration
}

func InitConfig(jobConfigFilePath string) *JobConfig {
	f, err := os.OpenFile(jobConfigFilePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var cfg JobConfig
	cfg.configFile = path.Base(jobConfigFilePath)
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}
