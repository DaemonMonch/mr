package mr

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"
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

func (j *Job) GetEnvs() ([]string, error) {
	if len(j.Envs) > 0 {
		return j.Envs, nil
	}
	f, err := os.OpenFile(j.EnvFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	var envs []string
	for sc.Scan() {
		l := strings.TrimFunc(sc.Text(), func(r rune) bool { return r == rune(' ') || r == rune('\n') || r == rune('\r') })
		if sc.Text() != "" && strings.IndexByte(l, '#') != 0 {
			envs = append(envs, sc.Text())
		}

	}
	if len(envs) > 0 {
		j.Envs = envs
	}
	return j.Envs, nil
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
