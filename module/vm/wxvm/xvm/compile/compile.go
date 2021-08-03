/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package compile

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Config configures the compiler
type Config struct {
	WxDecPath string
	OptLevel  int
}

// CompileCSource compile a wasm file to a c source file
func CompileCSource(cfg *Config, target, source string) (err error) {
	targetFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		targetFile.Close()
		if err != nil {
			os.Remove(target)
		}
	}()
	stderr := new(bytes.Buffer)
	cmd := exec.Command(cfg.WxDecPath, source)
	cmd.Stderr = stderr
	cmd.Stdout = targetFile
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("run wxdec error:%s %s", err, stderr.Bytes())
	}

	return nil
}

// CompileNativeLibrary compile a wasm file to native shared library
func CompileNativeLibrary(cfg *Config, target, source string) error {
	var err error
	if cfg.OptLevel < 0 || cfg.OptLevel > 2 {
		return errors.New("bad OptLevel, must in range [0,2]")
	}
	tmpdir, err := ioutil.TempDir("", "wxvm-compile")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	csource := filepath.Join(tmpdir, "wasm.c")
	err = CompileCSource(cfg, csource, source)
	if err != nil {
		return err
	}
	cheader := filepath.Join(tmpdir, "wasm-rt.h")
	//TODO:check if there is question when file permission is 0600
	err = ioutil.WriteFile(cheader, wasmRTHeader, 0600)
	if err != nil {
		return err
	}

	stderr := new(bytes.Buffer)
	cmd := exec.Command("cc", "-shared", "-fPIC",
		"-std=c99",
		"-O"+strconv.Itoa(cfg.OptLevel),
		"-o"+target,
		"-I.",
		"-I"+tmpdir,
		csource,
		"-lm",
	)
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("run cc error:%s", stderr.Bytes())
	}
	return nil
}
