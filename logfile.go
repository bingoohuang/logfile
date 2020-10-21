package logfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File 定义了写入日志的结构.
type File struct {
	// Pattern 定义了日志文件的布局，例如：/var/logs/{APP}/YYYYMMDD/{APP}_YYYYMMDD_{IP}_ZONE.log.
	Pattern string
	// MaxDelayDays 定义了日志时间落后于当前系统时间的最大天数，默认1天（也就是各系统产生的日志，必须在1天之内写入）.
	MaxDelayDays int
	// ArchiveDays 定义了多少天之前的日志文件，进行归档。0时不归档，否则必须大于MaxDelayDays.
	ArchiveDays int
	// DeleteArchiveDays 定义了多少天之前的归档日志删除。0时不删除。否则必须大于ArchiveDays.
	DeleteArchiveDays int
	// 是否写入后刷盘.
	Flush bool

	cache map[string]*os.File
}

// ErrOverMaxDelayDays 定义了写入日志的时间超过了MaxDelayDays的错误.
var ErrOverMaxDelayDays = errors.New("over max delay days")

// Write 写入一条日志.
func (f *File) Write(properties map[string]string, logTime time.Time, s string) error {
	logFile, err := f.createFile(properties, logTime)
	if err != nil {
		return err
	}

	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}

	if _, err := logFile.WriteString(s); err != nil {
		return err
	}

	if f.Flush {
		if err := logFile.Sync(); err != nil {
			return err
		}
	}

	return nil
}

func (f *File) createFile(properties map[string]string, t time.Time) (*os.File, error) {
	fn := f.Pattern
	for k, v := range properties {
		fn = strings.ReplaceAll(fn, "{"+k+"}", v)
	}

	fn = strings.ReplaceAll(fn, "YYYY", t.Format("2006"))
	fn = strings.ReplaceAll(fn, "MM", t.Format("01"))
	fn = strings.ReplaceAll(fn, "DD", t.Format("02"))

	if f.cache == nil {
		f.cache = make(map[string]*os.File)
	}

	if logFile, ok := f.cache[fn]; ok {
		return logFile, nil
	}

	if err := os.MkdirAll(filepath.Dir(fn), os.ModePerm); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	f.cache[fn] = logFile

	return logFile, nil
}
