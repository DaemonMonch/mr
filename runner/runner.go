package runner

import "context"

type Runner interface {
	Start(ctx context.Context,jobCfg *Job) error
	Wait() error
}