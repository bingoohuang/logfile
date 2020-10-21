package logfile_test

import (
	"github.com/bingoohuang/logfile"
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	l := logfile.File{
		Pattern:           "logs/{APP}/YYYYMMDD/{APP}_YYYYMMDD_{IP}_{ZONE}.log",
		MaxDelayDays:      1,    // 日志时间最多落后于当前系统时间1天
		ArchiveDays:       7,    // 归档7天前的日志
		DeleteArchiveDays: 90,   // 删除90天之前的归档日志
		Flush:             true, // 测试用，生产建议不打开，影响写入性能
	}

	day1, _ := time.Parse("2006-01-02 15:04:05", "2020-10-21 18:00:54")

	l.Write(map[string]string{
		"APP":  "ids",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day1, "我是第1天的一行日志，啦啦啦啦啦")

	day2 := day1.Add(24 * time.Hour)

	l.Write(map[string]string{
		"APP":  "ids",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day2, "我是第2天的一行日志，啦啦啦啦啦")
}
