//go:build windows

package callr

import (
	"fmt"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

func setCmdSysProcAttr(_ *exec.Cmd) {
	// No-op on Windows
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Only kill the parent on Windows
	_ = cmd.Process.Kill()
}

//nolint:gochecknoglobals // required for Job Object reuse
var (
	job         windows.Handle
	jobInitOnce sync.Once
	errJobInit  error
)

func assignProcessToJobObject(cmd *exec.Cmd) error {
	const processAllAccess = 0x1F0FFF

	jobInitOnce.Do(func() {
		job, errJobInit = windows.CreateJobObject(nil, nil)
		if errJobInit != nil {
			return
		}

		var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

		_, errJobInit = windows.SetInformationJobObject(
			job,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
	})

	if errJobInit != nil {
		return fmt.Errorf("failed to create job object: %w", errJobInit)
	}

	//nolint:gosec // safe: Pid is always >= 0
	processHandle, err := windows.OpenProcess(processAllAccess, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("failed to open process: %w", err)
	}
	defer func() { _ = windows.CloseHandle(processHandle) }()

	if err = windows.AssignProcessToJobObject(job, processHandle); err != nil {
		return fmt.Errorf("failed to assign process to job object: %w", err)
	}

	return nil
}
