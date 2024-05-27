package mr

import (
	"io"
	"log"
	"os"
	"sync"

	"github.com/gookit/color"
)

func GetLogger(configFile string, job *Job) io.Writer {
	return newConsoleLog(configFile + "-" + job.Name)
}

type renderFunc func(v ...interface{}) string

var (
	colorRenders = []renderFunc{color.FgGray.Render, color.FgGreen.Render, color.FgLightBlue.Render, color.FgLightCyan.Render, color.FgLightGreen.Render, color.FgLightYellow.Render,color.}
	logIdx       = 0
	m            = &sync.Mutex{}
)

func nextColorRender() renderFunc {
	
	m.Lock()
	defer m.Unlock()
	f := colorRenders[logIdx%len(colorRenders)]
	logIdx++
	return f
}

type consoleLog struct {
	log    *log.Logger
	prefix string
	render renderFunc
}

func newConsoleLog(logPrefix string) *consoleLog {
	log := log.New(os.Stdout, "", 0)
	l := &consoleLog{log: log, prefix: logPrefix, render: nextColorRender()}
	l.collapsePrefix()
	return l
}

func newConsoleLogForError(logPrefix string) *consoleLog {
	log := log.New(os.Stdout, "", 0)
	l := &consoleLog{log: log, prefix: logPrefix, render: color.FgRed.Render}
	l.collapsePrefix()
	return l
}

func (lw *consoleLog) Write(b []byte) (int, error) {
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

func (lw *consoleLog) collapsePrefix() {
	if len(lw.prefix) >= 20 {
		s := lw.prefix
		lw.prefix = s[:8] + "..." + s[len(s)-8:]
	}
}
