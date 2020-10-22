package logfile_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/bingoohuang/logfile"
	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	clockMock := clock.NewMock()
	l := logfile.File{
		Pattern:     "logs/{APP}/YYYYMMDD/{APP}_YYYYMMDD_{IP}_{ZONE}.log",
		ArchiveDays: 7,  // 归档7天前的日志
		DeleteDays:  90, // 删除90天之前的日志（包括归档日志）

		// 以下两项，是测试用，请在生产环境中忽略
		Flush: true,      // 测试用，生产建议不打开，影响写入性能
		Clock: clockMock, // 模拟时间
	}
	assert.Nil(t, l.Start())
	defer l.Close()

	day1, _ := time.Parse("2006-01-02 15:04:05", "2020-10-21 18:00:54")
	clockMock.Set(day1)

	assert.Nil(t, l.Write(map[string]string{
		"APP":  "ids",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day1, "我是第1天的一行日志，啦啦啦啦啦"))

	day2 := day1.Add(logfile.Day)

	assert.Nil(t, l.Write(map[string]string{
		"APP":  "ids",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day2, "我是第2天的一行日志，啦啦啦啦啦"))

	err := logfile.CreateTarGz("a.tar.gz", []string{
		"logs/ids/20201021/ids_20201021_192.168.0.1_zone01.log",
		"logs/ids/20201022/ids_20201022_192.168.0.1_zone01.log",
	})
	assert.Nil(t, err)

	os.RemoveAll("logs")

	r, _ := os.Open("a.tar.gz")
	assert.Nil(t, logfile.ExtractTarGz(r))
}

func TestArchiveDays(t *testing.T) {
	clockMock := clock.NewMock()
	day1, _ := time.Parse("2006-01-02 15:04:05", "2020-10-21 18:00:54")
	clockMock.Set(day1)

	l := logfile.File{
		Pattern:     "logs/{APP}/YYYYMMDD/{APP}_YYYYMMDD_{IP}_{ZONE}.log",
		ArchiveDays: 1, // 归档1天前的日志
		DeleteDays:  2, // 删除2天之前的日志（包括归档日志）

		// 以下两项，是测试用，请在生产环境中忽略
		Flush: true,      // 测试用，生产建议不打开，影响写入性能
		Clock: clockMock, // 模拟时间
	}
	assert.Nil(t, l.Start())
	defer l.Close()

	assert.Nil(t, l.Write(map[string]string{
		"APP":  "arhive",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day1, "我是第1天的一行日志，啦啦啦啦啦"))

	clockMock.Add(logfile.Day)
	gosched()

	day2 := day1.Add(logfile.Day)
	assert.Nil(t, l.Write(map[string]string{
		"APP":  "arhive",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day2, "我是第-1天的一行日志，啦啦啦啦啦"))

	clockMock.Add(logfile.Day)
	gosched()

	day2 = day2.Add(logfile.Day)

	assert.Nil(t, l.Write(map[string]string{
		"APP":  "arhive",
		"IP":   "192.168.0.1",
		"ZONE": "zone01",
	}, day2, "我是第-2天的一行日志，啦啦啦啦啦"))
}

func gosched() { time.Sleep(1 * time.Millisecond) }

/*
Benchmark_CaseInsensitiveReplace-12       521304              2316 ns/op
Benchmark_CaseSensitiveReplace-12       12651721                95.2 ns/op
Benchmark_Replace-12                     6340994               183 ns/op
*/

func Benchmark_CaseInsensitiveReplace(b *testing.B) {
	for n := 0; n < b.N; n++ {
		logfile.ReplaceIgnoreCase("{Title}|{Title}", "{title}", "My Title")
	}
}

func Benchmark_CaseSensitiveReplace(b *testing.B) {
	for n := 0; n < b.N; n++ {
		strings.ReplaceAll("{Title}|{Title}", "{Title}", "My Title")
	}
}

func Benchmark_ReplaceAll(b *testing.B) {
	for n := 0; n < b.N; n++ {
		logfile.ReplaceAll("{Title}|{Title}", "{Title}", "My Title")
	}
}
