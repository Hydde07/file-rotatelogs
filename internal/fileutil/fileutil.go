package fileutil

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lestrrat-go/strftime"
	"github.com/pkg/errors"
)

// GenerateFn creates a file name based on the pattern, the current time, and the
// rotation time.
//
// The bsase time that is used to generate the filename is truncated based
// on the rotation time.
func GenerateFn(pattern *strftime.Strftime, clock interface{ Now() time.Time }, rotationTime time.Duration) string {
	now := clock.Now()

	// XXX HACK: Truncate only happens in UTC semantics, apparently.
	// observed values for truncating given time with 86400 secs:
	//
	// before truncation: 2018/06/01 03:54:54 2018-06-01T03:18:00+09:00
	// after  truncation: 2018/06/01 03:54:54 2018-05-31T09:00:00+09:00
	//
	// This is really annoying when we want to truncate in local time
	// so we hack: we take the apparent local time in the local zone,
	// and pretend that it's in UTC. do our math, and put it back to
	// the local zone
	var base time.Time
	if now.Location() != time.UTC {
		base = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
		base = base.Truncate(rotationTime)
		base = time.Date(base.Year(), base.Month(), base.Day(), base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), base.Location())
	} else {
		base = now.Truncate(rotationTime)
	}

	return pattern.FormatString(base)
}

// CreateFile creates a new file in the given path, creating parent directories
// as necessary
func CreateFile(filename string) (*os.File, error) {
	// make sure the dir is existed, eg:
	// ./foo/bar/baz/hello.log must make sure ./foo/bar/baz is existed
	dirname := filepath.Dir(filename)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to create directory %s", dirname)
	}
	// if we got here, then we need to create a file
	fh, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, errors.Errorf("failed to open file %s: %s", filename, err)
	}

	return fh, nil
}

// CompressFile compresses a file
func CompressFile(filePath string, fileSuffix string, withTime bool) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var timeOnFile string
	if withTime {
		timeOnFile = time.Now().Format("2006-01-02_15-04")
		timeOnFile = fmt.Sprintf("-%s", timeOnFile)
	}
	if fileSuffix != "" {
		fileSuffix = fmt.Sprintf("-%s", fileSuffix)
	}

	f, err := os.Create(filePath + timeOnFile + fileSuffix + ".gz")
	if err != nil {
		return err
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	defer w.Close()

	_, err = w.Write(content)
	if err != nil {
		return err
	}

	err = os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}
