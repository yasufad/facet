package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modD3DCompiler = syscall.NewLazyDLL("d3dcompiler_47.dll")
	procD3DCompile = modD3DCompiler.NewProc("D3DCompile")
)

type id3d10BlobVtbl struct {
	QueryInterface   uintptr
	AddRef           uintptr
	Release          uintptr
	GetBufferPointer uintptr
	GetBufferSize    uintptr
}

type id3d10Blob struct {
	vtbl *id3d10BlobVtbl
}

func compileWithDLL(source []byte, entryPoint, target string) ([]byte, error) {
	if err := modD3DCompiler.Load(); err != nil {
		return nil, fmt.Errorf("load d3dcompiler_47.dll: %w", err)
	}

	entryPointC, err := syscall.BytePtrFromString(entryPoint)
	if err != nil {
		return nil, err
	}
	targetC, err := syscall.BytePtrFromString(target)
	if err != nil {
		return nil, err
	}

	var (
		codeBlob  *id3d10Blob
		errorBlob *id3d10Blob
	)

	const (
		d3dCompileOptimizationLevel3 = 1 << 15
		d3dCompileEnableStrictness   = 1 << 11
	)
	flags := uintptr(d3dCompileOptimizationLevel3 | d3dCompileEnableStrictness)

	srcPtr := uintptr(0)
	if len(source) > 0 {
		// Sound because source is kept live during SyscallN and not stored by D3DCompile.
		srcPtr = uintptr(unsafe.Pointer(&source[0]))
	}

	// Sound because pointers to local variables are passed to D3DCompile on the stack.
	hr, _, _ := procD3DCompile.Call(
		srcPtr,
		uintptr(len(source)),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(entryPointC)),
		uintptr(unsafe.Pointer(targetC)),
		flags,
		0,
		uintptr(unsafe.Pointer(&codeBlob)),
		uintptr(unsafe.Pointer(&errorBlob)),
	)

	if errorBlob != nil {
		// Sound because errorBlob vtable conforms to ID3D10Blob COM interface.
		bufPtr, _, _ := syscall.SyscallN(errorBlob.vtbl.GetBufferPointer, uintptr(unsafe.Pointer(errorBlob)))
		bufSize, _, _ := syscall.SyscallN(errorBlob.vtbl.GetBufferSize, uintptr(unsafe.Pointer(errorBlob)))
		errMsg := string(unsafe.Slice((*byte)(unsafe.Pointer(bufPtr)), int(bufSize)))
		syscall.SyscallN(errorBlob.vtbl.Release, uintptr(unsafe.Pointer(errorBlob)))
		if int32(hr) < 0 {
			return nil, fmt.Errorf("D3DCompile error (hr=0x%08x): %s", uint32(hr), strings.TrimSpace(errMsg))
		}
	}

	if int32(hr) < 0 || codeBlob == nil {
		return nil, fmt.Errorf("D3DCompile failed with hr=0x%08x", uint32(hr))
	}

	// Sound because codeBlob is a valid COM object returned by D3DCompile.
	bufPtr, _, _ := syscall.SyscallN(codeBlob.vtbl.GetBufferPointer, uintptr(unsafe.Pointer(codeBlob)))
	bufSize, _, _ := syscall.SyscallN(codeBlob.vtbl.GetBufferSize, uintptr(unsafe.Pointer(codeBlob)))
	compiled := make([]byte, int(bufSize))
	copy(compiled, unsafe.Slice((*byte)(unsafe.Pointer(bufPtr)), int(bufSize)))
	syscall.SyscallN(codeBlob.vtbl.Release, uintptr(unsafe.Pointer(codeBlob)))

	return compiled, nil
}

func compileWithFXC(fxcPath, srcFile, entryPoint, target, outFile string) error {
	cmd := exec.Command(fxcPath, "/T", target, "/E", entryPoint, "/Fo", outFile, "/O3", srcFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fxc failed: %w, output: %s", err, string(out))
	}
	return nil
}

type shaderJob struct {
	srcFile    string
	entryPoint string
	target     string
	outFile    string
}

func main() {
	shaderDir := filepath.Join("render", "d3d11", "shaders")
	if len(os.Args) > 1 {
		shaderDir = os.Args[1]
	}

	jobs := []shaderJob{
		{srcFile: "quad.hlsl", entryPoint: "vs_main", target: "vs_5_0", outFile: "quad_vs.dxbc"},
		{srcFile: "quad.hlsl", entryPoint: "ps_main", target: "ps_5_0", outFile: "quad_ps.dxbc"},
		{srcFile: "shadow.hlsl", entryPoint: "vs_main", target: "vs_5_0", outFile: "shadow_vs.dxbc"},
		{srcFile: "shadow.hlsl", entryPoint: "ps_main", target: "ps_5_0", outFile: "shadow_ps.dxbc"},
		{srcFile: "mono_sprite.hlsl", entryPoint: "vs_main", target: "vs_5_0", outFile: "mono_sprite_vs.dxbc"},
		{srcFile: "mono_sprite.hlsl", entryPoint: "ps_main", target: "ps_5_0", outFile: "mono_sprite_ps.dxbc"},
		{srcFile: "poly_sprite.hlsl", entryPoint: "vs_main", target: "vs_5_0", outFile: "poly_sprite_vs.dxbc"},
		{srcFile: "poly_sprite.hlsl", entryPoint: "ps_main", target: "ps_5_0", outFile: "poly_sprite_ps.dxbc"},
		{srcFile: "path.hlsl", entryPoint: "vs_main", target: "vs_5_0", outFile: "path_vs.dxbc"},
		{srcFile: "path.hlsl", entryPoint: "ps_main", target: "ps_5_0", outFile: "path_ps.dxbc"},
		{srcFile: "underline.hlsl", entryPoint: "vs_main", target: "vs_5_0", outFile: "underline_vs.dxbc"},
		{srcFile: "underline.hlsl", entryPoint: "ps_main", target: "ps_5_0", outFile: "underline_ps.dxbc"},
	}

	for _, job := range jobs {
		srcPath := filepath.Join(shaderDir, job.srcFile)
		outPath := filepath.Join(shaderDir, job.outFile)

		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			log.Fatalf("read shader %s: %v", srcPath, err)
		}

		bytecode, err := compileWithDLL(srcData, job.entryPoint, job.target)
		if err != nil {
			log.Fatalf("compile %s (%s, %s): %v", srcPath, job.entryPoint, job.target, err)
		}

		if err := os.WriteFile(outPath, bytecode, 0644); err != nil {
			log.Fatalf("write bytecode %s: %v", outPath, err)
		}

		fmt.Printf("Compiled %s (%s, %s) -> %s (%d bytes)\n", job.srcFile, job.entryPoint, job.target, job.outFile, len(bytecode))
	}
}
