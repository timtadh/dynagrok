package cmd

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"os"
	"strings"
	"compress/gzip"
)

func Input(input_path string) (reader io.Reader, closeall func(), err error) {
	stat, err := os.Stat(input_path)
	if err != nil {
		return nil, nil, err
	}
	if stat.IsDir() {
		return InputDir(input_path)
	} else {
		return InputFile(input_path)
	}
}

func InputFile(input_path string) (reader io.Reader, closeall func(), err error) {
	freader, err := os.Open(input_path)
	if err != nil {
		return nil, nil, err
	}
	if strings.HasSuffix(input_path, ".gz") {
		greader, err := gzip.NewReader(freader)
		if err != nil {
			freader.Close()
			return nil, nil, err
		}
		return greader, func() {
			greader.Close()
			freader.Close()
		}, nil
	}
	return freader, func() {
		freader.Close()
	}, nil
}

func InputDir(input_dir string) (reader io.Reader, closeall func(), err error) {
	var readers []io.Reader
	var closers []func()
	dir, err := ioutil.ReadDir(input_dir)
	if err != nil {
		return nil, nil, err
	}
	for _, info := range dir {
		if info.IsDir() {
			continue
		}
		creader, closer, err := InputFile(filepath.Join(input_dir, info.Name()))
		if err != nil {
			for _, closer := range closers {
				closer()
			}
			return nil, nil, err
		}
		readers = append(readers, creader)
		closers = append(closers, closer)
	}
	reader = io.MultiReader(readers...)
	return reader, func() {
		for _, closer := range closers {
			closer()
		}
	}, nil
}
