package mr

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func GetRunner() Runner {
	return new(PlainRunner)
}

type RunnerCtx struct {
	*Job
	log io.Writer
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

	cmd := exec.CommandContext(ctx, j.Cmd)
	cmd.Args = append(cmd.Args, j.Args...)
	cmd.Dir = j.Path
	envs, err := runnerCtx.Job.GetEnvs()
	if err != nil {
		fmt.Printf("%v", err)
	}
	cmd.Env = append(cmd.Environ(), envs...)
	cmd.Stdout = j.log
	cmd.Stderr = j.log //newConsoleLogForError(runnerCtx.Name)
	err = cmd.Start()
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	j.p = cmd.Process

	return nil
}
