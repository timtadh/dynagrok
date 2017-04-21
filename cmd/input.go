package cmd

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func Inputs(paths []string) (reader io.Reader, closeall func(), err error) {
	readers := make([]io.Reader, 0, len(paths))
	closers := make([]func(), 0, len(paths))
	for _, p := range paths {
		r, c, err := Input(p)
		if err != nil {
			return nil, nil, err
		}
		readers = append(readers, r)
		closers = append(closers, c)
	}
	reader = io.MultiReader(readers...)
	return reader, func() {
		for _, closer := range closers {
			closer()
		}
	}, nil
}

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
