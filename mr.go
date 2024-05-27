package mr

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

type runningJob struct {
	jobCfg        *Job
	jm            *JobManager
	jobId         int
	runner        Runner
	cancel        context.CancelFunc
	healthChecker HealthChecker

	stopOnce sync.Once
}

func (r *runningJob) start(ctx context.Context, configFile string) error {
	log := GetLogger(configFile, r.jobCfg)
	err := r.runner.Start(ctx, &RunnerCtx{r.jobCfg, log})
	if err != nil {
		return err
	}

	r.startHealthCheck(ctx)
	return nil
}

func (r *runningJob) startHealthCheck(ctx context.Context) {
	healthCheckCfg := r.jobCfg.HealthCheck
	var checkDelayTime time.Duration
	var checkInterval time.Duration
	if healthCheckCfg != nil {
		checkDelayTime = healthCheckCfg.CheckInterval
		checkInterval = healthCheckCfg.CheckInterval
	}

	doCheck := func() {
		err := r.healthChecker.Check()
		if err != nil {
			fmt.Printf("health check for %s fail %v\n", r.jobCfg.Name, err)
			r.jm.l.Lock()
			delete(r.jm.runningJobs, fmt.Sprint(r.jobId))
			r.jm.l.Unlock()
		} else {
			r.jm.l.Lock()
			r.jm.runningJobs[fmt.Sprint(r.jobId)] = r
			r.jm.l.Unlock()
		}
	}

	go func() {
		if checkDelayTime > 0 {
			<-time.After(healthCheckCfg.CheckDelayTime)
		}

		if checkInterval == 0 {
			doCheck()
			return
		}

		ticker := time.NewTicker(healthCheckCfg.CheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				doCheck()
			case <-ctx.Done():
				fmt.Printf("health check for %s stopped \n", r.jobCfg.Name)
				return
			}
		}
	}()
}

func (r *runningJob) stop() {
	r.stopOnce.Do(func() { r.cancel() })
}

func (r *runningJob) wait() {
	r.runner.Wait()
}

func (r *runningJob) stopAndWait() {
	r.stop()
	r.wait()
}

type JobManager struct {
	cfg         *JobConfig
	wg          *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	l           *sync.RWMutex
	runningJobs map[string]*runningJob
}

func NewJobManager(cfg *JobConfig) *JobManager {
	wg := &sync.WaitGroup{}
	wg.Add(len(cfg.Jobs))
	jm := &JobManager{cfg: cfg, wg: wg, l: &sync.RWMutex{}, runningJobs: make(map[string]*runningJob)}

	return jm
}

func (jm *JobManager) WaitTerminate() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	<-signalChan
	jm.cancel()

	jm.wg.Wait()
}

func (jm *JobManager) Jobs() []string {
	jm.l.RLock()
	defer jm.l.RUnlock()
	var r []string
	for k := range jm.runningJobs {
		j := jm.runningJobs[k]
		r = append(r, fmt.Sprintln(k, j.jobCfg.Name))
	}
	return r
}

func (jm *JobManager) Restart(id string) error {
	jm.l.RLock()
	runningjob := jm.runningJobs[id]
	jm.l.RUnlock()
	fmt.Printf("restart %s \n", runningjob.jobCfg.Name)
	jm.wg.Add(1)
	runningjob.stopAndWait()
	go func() {
		defer jm.wg.Done()
		nctx, cancel := context.WithCancel(jm.ctx)
		err := runningjob.start(nctx, jm.cfg.configFile)
		if err != nil {
			fmt.Printf("start %s fail %v\n", runningjob.jobCfg.Name, err)
			cancel()
			return
		}
		jm.l.Lock()
		runningjob.cancel = cancel
		jm.l.Unlock()
		runningjob.wait()
	}()

	return nil
}

func (jm *JobManager) StartJobs(ctx context.Context) {
	jm.ctx, jm.cancel = context.WithCancel(ctx)
	i := 0
	cfg := jm.cfg
	for jobName := range cfg.Jobs {
		job := cfg.Jobs[jobName]
		job.Name = jobName

		go func(id int) {
			defer jm.wg.Done()
			runner := GetRunner()
			nctx, cancel := context.WithCancel(jm.ctx)
			runningjob := &runningJob{jobCfg: job, runner: runner, cancel: cancel, jm: jm, jobId: id}
			runningjob.healthChecker = GetHealthChecker(job.HealthCheck)
			err := runningjob.start(nctx, jm.cfg.configFile)
			if err != nil {
				fmt.Printf("start %s fail %v\n", job.Name, err)
				cancel()
				return
			}
			runningjob.wait()

		}(i)
		i++

	}
}
