package gotex

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

var Verbose bool = false

type PipeResult struct {
	Reader io.ReadCloser
	Wait   func() error
}

func Download(url string) (*PipeResult, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	return &PipeResult{
		Reader: resp.Body,
		Wait: func() error {
			return nil
		},
	}, nil
}

func ReadFile(path string) (*PipeResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &PipeResult{
		Reader: f,
		Wait: func() error {
			return nil
		},
	}, nil
}

func ShellCmdDir(input io.Reader, dir string, name string, args ...string) (*PipeResult, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		_, _ = io.Copy(stdin, input)
		_ = stdin.Close()
	}()

	return &PipeResult{
		Reader: stdout,
		Wait: func() error {
			return cmd.Wait()
		},
	}, nil
}

func ShellCmd(input io.Reader, name string, args ...string) (*PipeResult, error) {
	return ShellCmdDir(input, "", name, args...)
}

func FileWriter(input io.Reader, target string) error {
	var w io.Writer
	var f *os.File
	var err error

	if target == "" || target == "-" {
		w = os.Stdout
	} else {
		f, err = os.Create(target)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	_, err = io.Copy(w, input)
	return err
}
