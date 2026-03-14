//go:build windows

package onboarding

func checkDiskSpace(_ string, _ uint64) (bool, error) {
	return true, nil
}
