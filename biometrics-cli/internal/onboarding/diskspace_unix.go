//go:build !windows

package onboarding

import "syscall"

func checkDiskSpace(path string, minBytes uint64) (bool, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return true, err
	}
	free := stat.Bavail * uint64(stat.Bsize)
	return free >= minBytes, nil
}
