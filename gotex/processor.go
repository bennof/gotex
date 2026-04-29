package gotex

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

const URL_TECTONIC = "https://drop-sh.fullyjustified.net"
const TECTONIC_BIN = "tectonic"
const LOCALE_TEXMF = "texmf"

const VAR_TECTONIC_CACHE_DIR = "TECTONIC_CACHE_DIR"
const TECTONIC_CACHE_FOLDER = ".tectonic-cache"
const VAR_GOTEX_PATH = "GOTEX_PATH"
const VAR_GOTEX_BIN = "GOTEX_BIN"
const VAR_GOTEX_TEXMF = "GOTEX_TEXMF"

type Processor struct {
	binpath  string
	treepath string

	searchpath []string
	binary     string
}

func NewProcessor(binpath, treepath string) (*Processor, error) {
	//check if binpath is valid
	err := checkBinary(binpath)
	if err != nil {
		return nil, err
	}

	//check if treepath is valid
	if ok, err := pathExists(treepath); !ok {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("TeXMF path not found: %s", treepath)
	}

	var searchpath []string
	err = filepath.WalkDir(filepath.Join(treepath, LOCALE_TEXMF), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			searchpath = append(searchpath, "-Z", "search-path="+path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect search paths: %w", err)
	}

	return &Processor{
		binpath:  binpath,
		treepath: treepath,

		searchpath: searchpath,
		binary:     filepath.Join(binpath, TECTONIC_BIN),
	}, nil
}

func checkBinary(binpath string) error {
	binary := filepath.Join(binpath, TECTONIC_BIN)
	if ok, err := pathExists(binary); !ok {
		if err != nil {
			return err
		}
		err = os.MkdirAll(binpath, os.ModePerm)
		if err != nil {
			return err
		}

		dl, err := Download(URL_TECTONIC)
		if err != nil {
			return err
		}
		defer dl.Reader.Close()

		sh, err := ShellCmdDir(dl.Reader, binpath, "sh")
		if err != nil {
			return err
		}
		defer sh.Reader.Close()

		if err := dl.Wait(); err != nil {
			return err
		}
		if err := sh.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func NewProcessorSimple() (*Processor, error) {
	var err error
	path := os.Getenv(VAR_GOTEX_PATH)
	binpath := os.Getenv(VAR_GOTEX_BIN)
	treepath := os.Getenv(VAR_GOTEX_TEXMF)
	if binpath == "" {
		if path != "" {
			binpath = path
		} else {
			binpath, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
	}

	if treepath == "" {
		if path != "" {
			treepath = path
		} else {
			treepath, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
	}

	return NewProcessor(binpath, treepath)
}

func (p *Processor) Process(input io.Reader, outdir string) (*PipeResult, error) {

	args := []string{"-p"}
	args = append(args, p.searchpath...)
	args = append(args, "-o", outdir, "-")

	cmd := exec.Command(
		p.binary,
		args...,
	)
	//cmd.Dir = p.treepath
	cmd.Env = append(os.Environ(),
		VAR_TECTONIC_CACHE_DIR+"="+filepath.Join(p.treepath, TECTONIC_CACHE_FOLDER),
	)
	cmd.Stdin = input

	if Verbose {
		fmt.Println("cmd:", cmd.Path, cmd.Args)
		fmt.Println("dir:", cmd.Dir)
		fmt.Println("env:", cmd.Env)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &PipeResult{
		Reader: stdout,
		Wait:   cmd.Wait,
	}, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
