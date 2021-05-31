package xvm

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/wxvm/xvm/compile"
	"chainmaker.org/chainmaker-go/wxvm/xvm/exec"
	"chainmaker.org/chainmaker-go/wxvm/xvm/runtime/emscripten"
	"chainmaker.org/chainmaker-go/wxvm/xvm/runtime/wasi"
	"errors"
	"golang.org/x/sync/singleflight"
	"io"
	"io/ioutil"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sync"
	"time"
)

const OptLevel = 0

type CodeManager struct {
	basedir string
	rundir  string

	makeCacheLock singleflight.Group

	mutex sync.Mutex
	codes map[string]exec.Code

	log *logger.CMLogger
}

func NewCodeManager(chainId string, basedir string) *CodeManager {
	runDirFull := filepath.Join(basedir)
	os.MkdirAll(basedir, 0755)

	return &CodeManager{
		basedir: basedir,
		rundir:  runDirFull,
		codes:   make(map[string]exec.Code),
		log:     logger.GetLoggerByChain(logger.MODULE_VM, chainId),
	}
}

func (c *CodeManager) lookupMemCache(keyId string) (exec.Code, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ccode, ok := c.codes[keyId]
	if !ok {
		return nil, false
	} else {
		return ccode, true
	}
}

func (c *CodeManager) lookupDiskCache(chainId string, contractId *commonPb.ContractId) (string, bool) {
	filePath := chainId + protocol.ContractStoreSeparator + contractId.ContractName
	fileName := contractId.ContractVersion + ".so"
	libPath := filepath.Join(c.basedir, filePath, fileName)
	if !fileExists(libPath) {
		return "", false
	}
	return libPath, true
}

func (c *CodeManager) makeDiskCache(chainId string, contractId *commonPb.ContractId, codebuf []byte) (string, error) {
	startTime := time.Now()
	filePath := chainId + protocol.ContractStoreSeparator + contractId.ContractName
	fileName := contractId.ContractVersion + ".so"
	basePath := filepath.Join(c.basedir, filePath)
	libPath := filepath.Join(c.basedir, filePath, fileName)

	os.MkdirAll(basePath, 0755)
	if err := c.CompileCode(codebuf, libPath); err != nil {
		c.log.Errorf("failed to compile wxvm code for contract %s", contractId.ContractName, err.Error())
		return "", err
	}
	c.log.Infof("compile wxvm code for contract %s,  time used %v", contractId.ContractName, time.Since(startTime))
	return libPath, nil
}

func (c *CodeManager) makeMemCache(contractKeyId string, libPath string,
	contextService *ContextService) (exec.Code, error) {

	c.mutex.Lock()
	defer c.mutex.Unlock()

	execCode, err := c.MakeExecCode(libPath, contextService)
	if err != nil {
		return nil, err
	}
	c.codes[contractKeyId] = execCode

	return execCode, nil
}
func (c *CodeManager) GetExecCode(chainId string, contractId *commonPb.ContractId,
	byteCode []byte, contextService *ContextService) (exec.Code, error) {

	contractKeyId := chainId + protocol.ContractStoreSeparator +
		contractId.ContractName + protocol.ContractStoreSeparator +
		contractId.ContractVersion

	execCode, ok := c.lookupMemCache(contractKeyId)
	if ok {
		return execCode, nil
	}

	// Only allow one goroutine make disk and memory cache at given contract name
	// other goroutine will block on the same contract name.
	icode, err, _ := c.makeCacheLock.Do(contractKeyId, func() (interface{}, error) {
		defer c.makeCacheLock.Forget(contractKeyId)
		execCode, ok := c.lookupMemCache(contractKeyId)
		if ok {
			return execCode, nil
		}
		var libPath string
		var err error
		if libPath, ok = c.lookupDiskCache(chainId, contractId); !ok {
			if libPath, err = c.makeDiskCache(chainId, contractId, byteCode); err != nil {
				return nil, err
			}
		}

		return c.makeMemCache(contractKeyId, libPath, contextService)
	})
	if err != nil {
		return nil, err
	}
	return icode.(exec.Code), nil
}

func (c *CodeManager) CompileCode(buf []byte, outputPath string) error {
	tmpdir, err := ioutil.TempDir("", "wxvm-compile")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	wasmpath := filepath.Join(tmpdir, "code.wasm")
	err = ioutil.WriteFile(wasmpath, buf, 0600)
	if err != nil {
		return err
	}

	libpath := filepath.Join(tmpdir, "code.so")

	wxDecPath, err := lookupWxDec()
	if err != nil {
		return err
	}

	cfg := &compile.Config{
		WxDecPath: wxDecPath,
		OptLevel:  OptLevel,
	}
	err = compile.CompileNativeLibrary(cfg, libpath, wasmpath)
	if err != nil {
		return err
	}
	if written, err := cpfile(outputPath, libpath); err != nil {
		return err
	} else if written == 0 {
		return errors.New("failed to copy file while compile native library for wxvm")
	} else {
		return nil
	}
}

func (c *CodeManager) MakeExecCode(libpath string, contextService *ContextService) (exec.Code, error) {
	resolvers := []exec.Resolver{
		emscripten.NewResolver(),
		NewContextServiceResolver(contextService),
		wasi.NewResolver(),
		BuiltinResolver,
	}
	resolver := exec.NewMultiResolver(
		resolvers...,
	)
	return exec.NewAOTCode(libpath, resolver)
}

func (c *CodeManager) RemoveCode(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.codes, name)
	os.RemoveAll(filepath.Join(c.basedir, name))
}

func lookupWxDec() (string, error) {
	wxdecBin := "wxdec"
	wxDecPath := filepath.Join(filepath.Dir(os.Args[0]), wxdecBin)
	stat, err := os.Stat(wxDecPath)
	if err == nil {
		if m := stat.Mode(); !m.IsDir() && m&0111 != 0 {
			return filepath.Abs(wxDecPath)
		}
	}
	// 再查找系统PATH目录
	return osexec.LookPath(wxdecBin)
}

func cpfile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func fileExists(fpath string) bool {
	stat, err := os.Stat(fpath)
	if err == nil && !stat.IsDir() {
		return true
	}
	return false
}
