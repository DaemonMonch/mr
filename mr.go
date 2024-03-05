package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"sync"

	"github.com/gookit/color"
	"github.com/mattn/go-tty"
	"github.com/mattn/go-tty/ttyutil"
	"gopkg.in/yaml.v3"
)

var (
	jobConfigFilePath = flag.String("f", "run.yml", "config file")
)

type JobConfig struct {
	Jobs       map[string]*Job
	configFile string
	output io.Writer
}

type Job struct {
	Cmd     string
	Args    []string
	Envs    []string
	EnvFile string `yaml:"env_file"`
	Path    string
	wg      *sync.WaitGroup
	stdout  io.Writer
	errout  io.Writer
	p *os.Process
}
func (j *Job) wait() {
	defer j.wg.Done()
	if j.p != nil {
		s,err := j.p.Wait()
		fmt.Printf("wait end %v %v \n",s,err)
	}
}

func (j *Job) Start(ctx context.Context) error {
	
	cmd := exec.CommandContext(ctx, j.Cmd)
	cmd.Args = append(cmd.Args, j.Args...)
	cmd.Dir = j.Path
	err := j.prepareEnv()
	if err != nil {
		fmt.Printf("%v", err)
	}
	cmd.Env = append(cmd.Environ(), j.Envs...)
	cmd.Stdout = j.stdout
	cmd.Stderr = j.errout
	err = cmd.Start()
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	j.p = cmd.Process

	return nil
}

func (j *Job) prepareEnv() error {
	if len(j.Envs) > 0 {
		return nil
	}
	f, err := os.OpenFile(j.EnvFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
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
	return nil
}

type renderFunc func(v ...interface{}) string

var colorRenders = []renderFunc{color.FgGray.Render, color.FgGreen.Render, color.FgLightBlue.Render, color.FgLightCyan.Render, color.FgLightGreen.Render, color.FgLightYellow.Render}

type logWriter struct {
	log    *log.Logger
	prefix string
	render renderFunc
}

func (lw *logWriter) Write(b []byte) (int, error) {
	s := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			lw.log.Printf("%-30s | %s\n", lw.render(lw.prefix), b[s:i])
			s = i + 1
		}
	}
	if len(b) > s {
		lw.log.Printf("%-30s | %s", lw.render(lw.prefix), b)
	}
	return len(b), nil
}

func (lw *logWriter) collapsePrefix() {
	if len(lw.prefix) >= 20 {
		s := lw.prefix
		lw.prefix = s[:8] + "..." + s[len(s) - 8:]
	}
}

func initConfig() *JobConfig {
	f, err := os.OpenFile(*jobConfigFilePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var cfg JobConfig
	cfg.configFile = path.Base(*jobConfigFilePath)
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}

type runningJob struct {
	job *Job
	name string
	cancel context.CancelFunc
}

type jobManager struct {
	cfg *JobConfig
	wg *sync.WaitGroup
	ctx context.Context

	l *sync.RWMutex
	runningJobs map[string]*runningJob
}

func newJobManager(cfg *JobConfig) *jobManager {
	wg := &sync.WaitGroup{}
	wg.Add(len(cfg.Jobs))
	jm := &jobManager{cfg: cfg,wg:wg,l: &sync.RWMutex{},runningJobs: make(map[string]*runningJob)}

	return jm
}

func (jm *jobManager) wait() {
	jm.wg.Wait()
}

func (jm *jobManager) jobs() []string{
	jm.l.RLock()
	defer jm.l.RUnlock()
	var r []string
	for k := range jm.runningJobs {
		j := jm.runningJobs[k]
		r = append(r, fmt.Sprintln(k,j.name))
	}
	return r
}

func (jm *jobManager) restart(id string) error {
	jm.l.RLock()
	runningjob := jm.runningJobs[id]
	jm.l.RUnlock()
	fmt.Println("restart id ",id)
	jm.wg.Add(1)
	runningjob.cancel()
	runningjob.job.wait()
	go func() {
		nctx,cancel := context.WithCancel(jm.ctx)
			err := runningjob.job.Start(nctx)
			if err != nil {
				fmt.Printf("start %s fail %v\n",id,err)
				cancel()
				return
			}
			jm.l.Lock()
			runningjob.cancel = cancel
			jm.l.Unlock()
			runningjob.job.wait()
	}()

	return nil
}

func (jm *jobManager) startJobs(ctx context.Context) {
	jm.ctx = ctx
	i := 0
	cfg := jm.cfg
	wg := jm.wg
	for j := range cfg.Jobs {
		jobName := j
		job := cfg.Jobs[j]
		log := log.New(cfg.output, "", 0)
		lw := &logWriter{log: log, prefix: cfg.configFile + "-" + j, render: colorRenders[i%len(colorRenders)]}
		lw.collapsePrefix()
		job.wg = wg
		job.stdout = lw
		job.errout = lw
		
		go func(id int) {
			nctx,cancel := context.WithCancel(ctx)
			err := job.Start(nctx)
			if err != nil {
				println(err)
				cancel()
				return
			}
			jm.l.Lock()
			runningjob := &runningJob{job,jobName,cancel}
			jm.runningJobs[fmt.Sprint(id)] = runningjob
			jm.l.Unlock()
			runningjob.job.wait()

		}(i)
		i++
		
	}
}

func main() {
	flag.Parse()
	cfg := initConfig()

	tty,err := tty.Open()

	if err != nil {
		panic(err)
	}
	cfg.output = tty.Output()

	jobManager := newJobManager(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	jobManager.startJobs(ctx)

	go func() {
		for {
			cmd,_ := ttyutil.ReadLine(tty)
			if cmd == "ls" {
				for _,j := range jobManager.jobs() {
					fmt.Fprintf(tty.Output(),"> %s",j)
				}
			}

			if strings.Contains(cmd,"rs")  {
				sp := strings.Split(cmd," ")
				if len(sp) == 2{
					jobManager.restart(sp[1])	
				}
			}
		}
		
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	<-signalChan
	cancel()

	jobManager.wait()
}
