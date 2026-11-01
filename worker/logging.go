package worker

import (
	"os"
)

const maxLogFileBytes = 256 << 10 // At most 256 KiB current + 256 KiB previous per account.

func (s *AccountState) appendFileLog(line string) error {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	if len(line) > maxLogFileBytes {
		line = line[:maxLogFileBytes-1] + "\n"
	}
	info, err := os.Stat(s.LogFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if info != nil && info.Size()+int64(len(line)) > maxLogFileBytes {
		// Remove the previous generation first for Windows rename semantics.
		if err := os.Remove(s.LogFile + ".1"); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(s.LogFile, s.LogFile+".1"); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(s.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteString(line)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
