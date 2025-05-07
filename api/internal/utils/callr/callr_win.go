//go:build windows

package callr

import (
	"fmt"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type winPlat struct {
	mu   sync.Mutex
	jobs map[int]windows.Handle
}

//nolint:gochecknoglobals // this is a singleton
var plat platform = &winPlat{jobs: make(map[int]windows.Handle)}

func (*winPlat) Setup(_ *exec.Cmd) error {
	// no-op
	return nil
}

func (w *winPlat) Assign(cmd *exec.Cmd) error {
	const processAllAccess = 0x1F0FFF

	// 1) create Job Object
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create job object: %w", err)
	}
	// configure it to kill all children
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job) // clean up on error
		return fmt.Errorf("failed to set job object information: %w", err)
	}

	// 2) assign the R process itself
	//nolint:gosec // PID is always > 0
	ph, err := windows.OpenProcess(processAllAccess, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("failed to open process: %w", err)
	}
	if err = windows.AssignProcessToJobObject(job, ph); err != nil {
		_ = windows.CloseHandle(ph)
		_ = windows.CloseHandle(job)
		return fmt.Errorf("failed to assign process to job object: %w", err)
	}
	_ = windows.CloseHandle(ph) // we no longer need this one

	// 3) remember the job handle so Teardown can find it
	w.mu.Lock()
	w.jobs[cmd.Process.Pid] = job
	w.mu.Unlock()
	return nil
}
func (w *winPlat) Teardown(cmd *exec.Cmd) error {
	pid := cmd.Process.Pid

	w.mu.Lock()
	job, ok := w.jobs[pid]
	if ok {
		delete(w.jobs, pid)
	}
	w.mu.Unlock()

	if !ok {
		// fallback: just kill the process
		err := cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		return nil
	}

	// this kills *all* procs in the Job (even if they detached)
	if err := windows.TerminateJobObject(job, 1); err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("failed to terminate job object: %w", err)
	}
	// now close the job handle for real
	_ = windows.CloseHandle(job)
	return nil
}
