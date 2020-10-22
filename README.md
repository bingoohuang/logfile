# logfile

write log from other systems to different files

1. 当前写入：日志（包含时间）=> `应用名/日期(YYYYMMDD)/应用名_YYYYMMDD_源IP_分区ID.log`
1. 超期归档：归档n（如7天）之前的日志：`应用名/日期(YYYYMMDD)/应用名_YYYYMMDD_源IP_分区ID.tar.gz` (保留.tar.gz，在不限制大小时只包含1一个文件，为后续可能的日志文件最大大小做预留)
1. 过期删除：删除n（如90天）之前的归档日志

usage:

```go
package main

import (
	"github.com/bingoohuang/logfile"
	"time"
)

func main() {
	l := logfile.File{
		Pattern:     "{APP}/YYYYMMDD/{APP}_YYYYMMDD_{IP}_{ZONE}.log",
		ArchiveDays: 1, // 归档1天前的日志
		DeleteDays:  2, // 删除2天之前的日志（包括归档日志）
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
```
