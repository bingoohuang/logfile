package logfile

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/sirupsen/logrus"
)

// File 定义了写入日志的结构.
type File struct {
	// Pattern 定义了日志文件的布局，例如：/var/logs/{APP}/YYYYMMDD/{APP}_YYYYMMDD_{IP}_ZONE.log.
	Pattern string
	// ArchiveDays 定义了多少天之前的日志文件，进行归档。0时不归档.
	ArchiveDays int
	// DeleteDays 定义了多少天之前的日志删除（包括归档日志）。0时不删除.
	DeleteDays int

	cacheLock sync.Mutex
	cache     map[string]*cacheValue
	Clock     clock.Clock
	stop      chan struct{}
	started   bool
}

type cacheValue struct {
	fn         string
	f          *os.File
	createTime time.Time
	properties map[string]string
	msgCh      chan string
	exitChan   chan struct{}
}

func (v *cacheValue) write(s string) {
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}

	v.msgCh <- s
}

func (v *cacheValue) run(f *File) {
	defer func() {
		v.exitChan <- struct{}{}
		f.cacheRemove(v.fn)
	}()

	for {
		msgCount, ok := v.writeUntilEmpty()
		if !ok {
			return
		}

		if msgCount > 0 {
			if err := v.f.Sync(); err != nil {
				log.Printf("sync file %s error: %v", v.fn, err)
				return
			}
		}
	}
}

func (v *cacheValue) writeUntilEmpty() (int, bool) {
	count := 0
	for {
		select {
		case s, ok := <-v.msgCh:
			if !ok {
				return count, false
			}
			if _, err := v.f.WriteString(s); err != nil {
				log.Printf("write file %s error: %v", v.fn, err)
				return count, false
			}
			count++
		default:
			return count, true
		}
	}
}

func (v *cacheValue) close() error {
	close(v.msgCh)
	<-v.exitChan
	return nil
}

// Start starts the logfile archiving and deleting works after necessary initialization. .
func (f *File) Start() error {
	f.cache = make(map[string]*cacheValue)
	if f.Clock == nil {
		f.Clock = clock.New()
	}

	f.stop = make(chan struct{})
	go f.schedule()

	f.started = true

	return nil
}

// Close shutdowns the scheduler.
func (f *File) Close() error {
	f.started = false
	f.stop <- struct{}{}

	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	for k, v := range f.cache {
		_ = v.close()
		delete(f.cache, k)
	}

	return nil
}

// ErrOverArchiveDays 定义了写入日志的时间超过了ArchiveDays的错误.
var ErrOverArchiveDays = errors.New("over max archive days")

// ErrNotStarted 表示没有调用Start方法开始，或者在Close之后继续Write.
var ErrNotStarted = errors.New("not started")

// Write 写入一条日志.
func (f *File) Write(properties map[string]string, logTime time.Time, s string) error {
	nowTime := f.Clock.Now()
	if logTime.Before(nowTime.Add(-Day * time.Duration(f.ArchiveDays))) {
		return ErrOverArchiveDays
	}

	if !f.started {
		return ErrNotStarted
	}

	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	v, err := f.createFile(properties, logTime)
	if err != nil {
		return err
	}

	v.write(s)

	return nil
}

func (f *File) createFile(properties map[string]string, t time.Time) (*cacheValue, error) {
	fn := f.createFileName(properties, t)

	if logFile, ok := f.cache[fn]; ok {
		return logFile, nil
	}

	if err := os.MkdirAll(filepath.Dir(fn), os.ModePerm); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	v := &cacheValue{
		fn:         fn,
		f:          logFile,
		createTime: f.Clock.Now(),
		properties: properties,
		msgCh:      make(chan string, 1000),
		exitChan:   make(chan struct{}),
	}
	go v.run(f)

	f.cache[fn] = v

	return v, nil
}

func (f *File) createFileName(properties map[string]string, t time.Time) string {
	fn := f.Pattern
	for k, v := range properties {
		fn = ReplaceAll(fn, "{"+k+"}", v)
	}

	fn = ReplaceAll(fn, "YYYY", t.Format("2006"))
	fn = ReplaceAll(fn, "MM", t.Format("01"))
	fn = ReplaceAll(fn, "DD", t.Format("02"))

	return fn
}

func (f *File) schedule() {
	tick := f.Clock.Ticker(Day)
	defer tick.Stop()

	logrus.Infof("scheduler started")
	defer logrus.Infof("scheduler stopped")

	for {
		select {
		case <-tick.C:
			f.clearOldFiles()

			if f.DeleteDays > 0 {
				f.deleteFiles()
			}
			if f.ArchiveDays > 0 {
				f.archiveFiles()
			}
		case <-f.stop:
			return
		}
	}
}

func (f *File) deleteFiles() {
	from := f.Clock.Now().Add(-Day * time.Duration(f.DeleteDays))
	for {
		// 没有找到需要删除的任何文件，结束
		if !f.iterateCache4Delete(from) {
			break
		}

		// 再往前推一天
		from = from.Add(-Day)
	}
}

func (f *File) iterateCache4Delete(from time.Time) bool {
	found := false

	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	for _, v := range f.cache {
		fn := f.createFileName(v.properties, from)
		matches, _ := filepath.Glob(fn + "*")
		found = found || len(matches) > 0
		removeFiles(matches)
	}

	return found
}

func removeFiles(matches []string) {
	for _, f := range matches {
		if err := os.Remove(f); err != nil {
			logrus.Warnf("remove %s error: %v", f, err)
		} else {
			logrus.Infof("remove %s success", f)
		}
	}
}

func (f *File) archiveFiles() {
	from := f.Clock.Now().Add(-Day * time.Duration(f.ArchiveDays))
	for {
		// 没有找到需要删除的任何文件，结束
		if !f.iterateCache4Archive(from) {
			break
		}

		// 再往前推一天
		from = from.Add(-Day)
	}
}

func (f *File) iterateCache4Archive(from time.Time) bool {
	found := false

	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	for _, v := range f.cache {
		fn := f.createFileName(v.properties, from)
		matches, _ := filepath.Glob(fn + "*")
		if matches = filterOutTarGz(matches); len(matches) == 0 {
			continue
		}

		found = true

		if err := CreateTarGz(fn+".tar.gz", matches); err != nil {
			logrus.Warnf("create %s.tar.gz with files %v error: %v", fn, matches, err)
		} else {
			logrus.Infof("create %s.tar.gz success with files %v", fn, matches)
			removeFiles(matches)
		}
	}

	return found
}

// Day means 24 hours.
const Day = time.Hour * 24

func (f *File) clearOldFiles() {
	now := f.Clock.Now()

	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	for k, v := range f.cache {
		if now.Sub(v.createTime) > 2*Day {
			_ = v.close()
			delete(f.cache, k)
		}
	}
}

func (f *File) cacheRemove(k string) {
	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	delete(f.cache, k)
}

func filterOutTarGz(matches []string) []string {
	if len(matches) == 0 {
		return matches
	}

	r := make([]string, 0, len(matches))

	for _, v := range matches {
		if !strings.HasSuffix(v, ".tar.gz") {
			r = append(r, v)
		}
	}

	return r
}

// ReplaceIgnoreCase replaces  all the search string to replace in subject with case-insensitive.
func ReplaceIgnoreCase(subject string, search string, replace string) string {
	r := regexp.MustCompile("(?i)" + regexp.QuoteMeta(search))
	return r.ReplaceAllString(subject, replace)
}

// ReplaceAll replaces  all the search string to replace in subject with case-insensitive.
func ReplaceAll(subject string, search string, replace string) string {
	u := strings.ToUpper(subject)
	s := strings.ToUpper(search)
	r := ""
	l := len(s)

	for {
		i := strings.Index(u, s)
		if i < 0 {
			return r + subject
		}

		r += subject[:i] + replace
		u = u[i+l:]
		subject = subject[i+l:]
	}
}
