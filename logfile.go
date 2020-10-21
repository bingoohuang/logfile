package logfile

import (
	"errors"
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
}

// ErrOverMaxDelayDays 定义了写入日志的时间超过了MaxDelayDays的错误.
var ErrOverMaxDelayDays = errors.New("over max delay days")

// Write 写入一条日志.
func (*File) Write(properties map[string]string, logTime time.Time, logContent string) error {
	return nil
}
