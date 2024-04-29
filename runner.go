package mr

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/gookit/color"
)

func GetRunner() Runner {
	return new(PlainRunner)
}

type RunnerCtx struct {
	*Job
	configFile string
}

type Runner interface {
	Start(ctx context.Context, runnerCtx *RunnerCtx) error
	Wait() error
}

type PlainRunner struct {
	*RunnerCtx
	p *os.Process
}

func (j *PlainRunner) Wait() error {
	if j.p != nil {
		_, err := j.p.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *PlainRunner) Start(ctx context.Context, runnerCtx *RunnerCtx) error {
	j.RunnerCtx = runnerCtx
	log := log.New(os.Stdout, "", 0)
	stdout := &logWriter{log: log, prefix: runnerCtx.configFile + "-" + runnerCtx.Name, render: colorRenders[rand.Int()%len(colorRenders)]}
	stdout.collapsePrefix()

	errout := &logWriter{log: log, prefix: runnerCtx.configFile + "-" + runnerCtx.Name, render: color.FgRed.Render}
	errout.collapsePrefix()

	cmd := exec.CommandContext(ctx, j.Cmd)
	cmd.Args = append(cmd.Args, j.Args...)
	cmd.Dir = j.Path
	err := j.prepareEnv()
	if err != nil {
		fmt.Printf("%v", err)
	}
	cmd.Env = append(cmd.Environ(), j.Envs...)
	cmd.Stdout = stdout
	cmd.Stderr = errout
	err = cmd.Start()
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	j.p = cmd.Process

	return nil
}

func (j *PlainRunner) prepareEnv() error {
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
			lw.log.Printf("%-30s | %s\n", lw.render(lw.prefix), lw.render(string(b[s:i])))
			s = i + 1
		}
	}
	if len(b) > s {
		lw.log.Printf("%-30s | %s", lw.render(lw.prefix), lw.render(string(b)))
	}
	return len(b), nil
}

func (lw *logWriter) collapsePrefix() {
	if len(lw.prefix) >= 20 {
		s := lw.prefix
		lw.prefix = s[:8] + "..." + s[len(s)-8:]
	}
}
