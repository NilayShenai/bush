package jobcontrol

import (
	"bush/internal/color"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
)

type JobStatus string

const (
	StatusRunning   JobStatus = "Running"
	StatusStopped   JobStatus = "Stopped"
	StatusCompleted JobStatus = "Done"
	StatusFailed    JobStatus = "Failed"
)

type Job struct {
	ID       int
	PID      int
	CmdStr   string
	Cmd      *exec.Cmd
	Status   JobStatus
	ExitCode int
}

type Manager struct {
	jobs   map[int]*Job
	nextID int
	mu     sync.Mutex
}

var DefaultManager = &Manager{
	jobs:   make(map[int]*Job),
	nextID: 1,
}

func (m *Manager) AddJob(cmd *exec.Cmd, cmdStr string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	job := &Job{
		ID:     m.nextID,
		PID:    cmd.Process.Pid,
		CmdStr: cmdStr,
		Cmd:    cmd,
		Status: StatusRunning,
	}
	m.jobs[m.nextID] = job
	m.nextID++

	fmt.Printf("[%d] %d\n", job.ID, job.PID)

	go func(j *Job) {
		err := j.Cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				j.ExitCode = exitErr.ExitCode()
			} else {
				j.ExitCode = 1
			}
			j.Status = StatusFailed
		} else {
			j.ExitCode = 0
			j.Status = StatusCompleted
		}
	}(job)

	return job
}

func (m *Manager) CheckJobs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, job := range m.jobs {
		if job.Status == StatusCompleted || job.Status == StatusFailed {
			statusStr := color.Colorize(string(job.Status), color.PastelMint)
			if job.Status == StatusFailed {
				statusStr = color.Colorize(string(job.Status), color.PastelRed)
			}
			fmt.Printf("[%d]+ %s \t%s\n", id, statusStr, job.CmdStr)
			delete(m.jobs, id)
		}
	}
}

func (m *Manager) ListJobs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.jobs) == 0 {
		fmt.Println(color.Colorize("No active background jobs", color.PastelGray))
		return
	}

	for id, job := range m.jobs {
		statusColor := color.PastelMint
		if job.Status == StatusRunning {
			statusColor = color.Lavender
		} else if job.Status == StatusStopped {
			statusColor = color.PastelPeach
		}
		fmt.Printf("[%d] %s \t%s (pid: %d)\n",
			id,
			color.Colorize(string(job.Status), statusColor),
			job.CmdStr,
			job.PID,
		)
	}
}

func (m *Manager) GetJob(id int) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[id]
}

func (m *Manager) Foreground(id int) error {
	m.mu.Lock()
	job, exists := m.jobs[id]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("job %d not found", id)
	}

	fmt.Printf("%s\n", job.CmdStr)

	if err := syscall.Kill(job.PID, syscall.SIGCONT); err != nil {
		return err
	}

	var status syscall.WaitStatus
	_, err := syscall.Wait4(job.PID, &status, 0, nil)

	m.mu.Lock()
	delete(m.jobs, id)
	m.mu.Unlock()

	return err
}

func (m *Manager) Background(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}

	if err := syscall.Kill(job.PID, syscall.SIGCONT); err != nil {
		return err
	}

	job.Status = StatusRunning
	fmt.Printf("[%d]+ %s &\n", job.ID, job.CmdStr)
	return nil
}

func SetupSignals() {

}
