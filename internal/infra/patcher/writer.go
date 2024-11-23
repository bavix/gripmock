package patcher

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync/atomic"
)

var (
	optionGoPackageRegexp = regexp.MustCompile("option go_package.+;\n?")
	syntaxRegexp          = regexp.MustCompile("syntax.+;\n?")
	ErrSyntaxNotFound     = errors.New("proto syntax not found")
)

type fileWriterWrapper struct {
	writer      io.Writer
	packageName string
	checked     uint32
}

func NewWriterWrapper(writer io.Writer, packageName string) io.Writer {
	return &fileWriterWrapper{writer: writer, packageName: packageName}
}

func (f *fileWriterWrapper) Write(p []byte) (int, error) {
	if atomic.LoadUint32(&f.checked) == 1 {
		return f.writer.Write(p)
	}

	atomic.StoreUint32(&f.checked, 1)

	const (
		syntaxIndexes = 2
		prefix        = "github.com/bavix/gripmock"
	)

	n := len(p)
	goPackage := []byte(fmt.Sprintf("option go_package = \"%s/%s\";\n", prefix, f.packageName))

	if optionGoPackageRegexp.Match(p) {
		_, err := f.writer.Write(optionGoPackageRegexp.ReplaceAll(p, goPackage))

		return n, err
	}

	indexes := syntaxRegexp.FindIndex(p)
	if len(indexes) != syntaxIndexes {
		return 0, ErrSyntaxNotFound
	}

	_, err := f.writer.Write(
		append(p[:indexes[1]], append(goPackage, p[indexes[1]:]...)...),
	)

	return n, err
}
