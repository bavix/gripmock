package patcher

import (
	"errors"
	"fmt"
	"io"
	"regexp"
)

var (
	optionGoPackageRegexp = regexp.MustCompile("option go_package.+;\n?")
	syntaxRegexp          = regexp.MustCompile("syntax.+;\n?")
	ErrSyntaxNotFound     = errors.New("proto syntax not found")
)

type fileWriterWrapper struct {
	writer      io.Writer
	packageName string
}

func NewWriterWrapper(writer io.Writer, packageName string) io.Writer {
	return &fileWriterWrapper{writer: writer, packageName: packageName}
}

func (f *fileWriterWrapper) Write(p []byte) (int, error) {
	const (
		syntaxIndexes = 2
		prefix        = "github.com/bavix/gripmock/protogen"
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
