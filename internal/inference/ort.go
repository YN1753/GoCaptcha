package inference

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	inputName  = "input1"
	outputName = "387"
)

type Result struct {
	Data     []float32
	Sequence int
	Batch    int
	Classes  int
}

type Engine struct {
	handle     uintptr
	api        uintptr
	env        uintptr
	options    uintptr
	session    uintptr
	memoryInfo uintptr
	logID      []byte
	closeOnce  sync.Once
	closeErr   error
}

func New(libraryPath string, model []byte) (*Engine, error) {
	if libraryPath == "" {
		return nil, errors.New("inference: empty ONNX Runtime library path")
	}
	if len(model) == 0 {
		return nil, errors.New("inference: empty model")
	}

	handle, err := purego.Dlopen(libraryPath, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, fmt.Errorf("inference: load ONNX Runtime %q: %w", libraryPath, err)
	}

	engine := &Engine{handle: handle}
	ready := false
	defer func() {
		if !ready {
			_ = engine.Close()
		}
	}()

	symbol, err := purego.Dlsym(handle, "OrtGetApiBase")
	if err != nil {
		return nil, fmt.Errorf("inference: find OrtGetApiBase: %w", err)
	}
	base, _, _ := purego.SyscallN(symbol)
	if base == 0 {
		return nil, errors.New("inference: OrtGetApiBase returned nil")
	}

	getAPI := *(*uintptr)(cPointer(base + ortBaseGetAPI))
	api, _, _ := purego.SyscallN(getAPI, ortAPIVersion)
	if api == 0 {
		return nil, fmt.Errorf("inference: ONNX Runtime does not provide C API version %d", ortAPIVersion)
	}
	engine.api = api

	getVersion := *(*uintptr)(cPointer(base + ortBaseGetVersion))
	versionPointer, _, _ := purego.SyscallN(getVersion)
	version := cString(versionPointer)
	if !strings.HasPrefix(version, "1.23.2") {
		return nil, fmt.Errorf("inference: unsupported ONNX Runtime version %q, want 1.23.2", version)
	}

	engine.logID = append([]byte("gocaptcha"), 0)
	var env uintptr
	if err := engine.check("CreateEnv", engine.call(ortAPICreateEnv, 3, uintptr(unsafe.Pointer(&engine.logID[0])), uintptr(unsafe.Pointer(&env)))); err != nil {
		return nil, err
	}
	engine.env = env

	var options uintptr
	if err := engine.check("CreateSessionOptions", engine.call(ortAPICreateOptions, uintptr(unsafe.Pointer(&options)))); err != nil {
		return nil, err
	}
	engine.options = options
	if err := engine.check("SetSessionGraphOptimizationLevel", engine.call(ortAPISetGraphOptimization, options, 99)); err != nil {
		return nil, err
	}
	if err := engine.check("SetIntraOpNumThreads", engine.call(ortAPISetIntraOpThreads, options, 1)); err != nil {
		return nil, err
	}

	var memoryInfo uintptr
	if err := engine.check("CreateCpuMemoryInfo", engine.call(ortAPICreateCPUMemInfo, 0, 0, uintptr(unsafe.Pointer(&memoryInfo)))); err != nil {
		return nil, err
	}
	engine.memoryInfo = memoryInfo

	var session uintptr
	status := engine.call(
		ortAPICreateSession,
		env,
		uintptr(unsafe.Pointer(&model[0])),
		uintptr(len(model)),
		options,
		uintptr(unsafe.Pointer(&session)),
	)
	runtime.KeepAlive(model)
	if err := engine.check("CreateSessionFromArray", status); err != nil {
		return nil, err
	}
	engine.session = session

	ready = true
	return engine, nil
}

func (e *Engine) Run(input []float32, height, width int) (*Result, error) {
	if e == nil || e.session == 0 {
		return nil, errors.New("inference: engine is closed")
	}
	if height != 64 {
		return nil, fmt.Errorf("inference: input height must be 64, got %d", height)
	}
	if width < 1 || len(input) != height*width {
		return nil, fmt.Errorf("inference: input length %d does not match %dx%d", len(input), height, width)
	}

	shape := []int64{1, 1, int64(height), int64(width)}
	var inputValue uintptr
	status := e.call(
		ortAPICreateTensor,
		e.memoryInfo,
		uintptr(unsafe.Pointer(&input[0])),
		uintptr(len(input)*4),
		uintptr(unsafe.Pointer(&shape[0])),
		uintptr(len(shape)),
		1,
		uintptr(unsafe.Pointer(&inputValue)),
	)
	if err := e.check("CreateTensorWithDataAsOrtValue", status); err != nil {
		return nil, err
	}
	defer e.call(ortAPIReleaseValue, inputValue)

	inputNameBytes := append([]byte(inputName), 0)
	outputNameBytes := append([]byte(outputName), 0)
	inputNames := []uintptr{uintptr(unsafe.Pointer(&inputNameBytes[0]))}
	outputNames := []uintptr{uintptr(unsafe.Pointer(&outputNameBytes[0]))}
	inputs := []uintptr{inputValue}
	outputs := []uintptr{0}

	status = e.call(
		ortAPIRun,
		e.session,
		0,
		uintptr(unsafe.Pointer(&inputNames[0])),
		uintptr(unsafe.Pointer(&inputs[0])),
		1,
		uintptr(unsafe.Pointer(&outputNames[0])),
		1,
		uintptr(unsafe.Pointer(&outputs[0])),
	)
	runtime.KeepAlive(input)
	runtime.KeepAlive(shape)
	runtime.KeepAlive(inputNameBytes)
	runtime.KeepAlive(outputNameBytes)
	runtime.KeepAlive(inputNames)
	runtime.KeepAlive(outputNames)
	runtime.KeepAlive(inputs)
	runtime.KeepAlive(outputs)
	if err := e.check("Run", status); err != nil {
		return nil, err
	}

	output := outputs[0]
	if output == 0 {
		return nil, errors.New("inference: ONNX Runtime returned no output")
	}
	defer e.call(ortAPIReleaseValue, output)

	var typeInfo uintptr
	if err := e.check("GetTensorTypeAndShape", e.call(ortAPIGetTypeAndShape, output, uintptr(unsafe.Pointer(&typeInfo)))); err != nil {
		return nil, err
	}
	defer e.call(ortAPIReleaseTypeShape, typeInfo)

	var dimensionCount uintptr
	if err := e.check("GetDimensionsCount", e.call(ortAPIGetDimensionNum, typeInfo, uintptr(unsafe.Pointer(&dimensionCount)))); err != nil {
		return nil, err
	}
	if dimensionCount != 3 {
		return nil, fmt.Errorf("inference: unexpected output rank %d, want 3", dimensionCount)
	}

	dimensions := make([]int64, dimensionCount)
	if err := e.check("GetDimensions", e.call(ortAPIGetDimensions, typeInfo, uintptr(unsafe.Pointer(&dimensions[0])), dimensionCount)); err != nil {
		return nil, err
	}

	var dataPointer uintptr
	if err := e.check("GetTensorMutableData", e.call(ortAPIGetTensorData, output, uintptr(unsafe.Pointer(&dataPointer)))); err != nil {
		return nil, err
	}

	total := 1
	for _, dimension := range dimensions {
		if dimension <= 0 {
			return nil, fmt.Errorf("inference: invalid output dimension %d", dimension)
		}
		total *= int(dimension)
	}
	values := unsafe.Slice((*float32)(cPointer(dataPointer)), total)
	copied := append([]float32(nil), values...)

	return &Result{
		Data:     copied,
		Sequence: int(dimensions[0]),
		Batch:    int(dimensions[1]),
		Classes:  int(dimensions[2]),
	}, nil
}

func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	e.closeOnce.Do(func() {
		if e.session != 0 {
			e.call(ortAPIReleaseSession, e.session)
			e.session = 0
		}
		if e.memoryInfo != 0 {
			e.call(ortAPIReleaseMemInfo, e.memoryInfo)
			e.memoryInfo = 0
		}
		if e.options != 0 {
			e.call(ortAPIReleaseSessionOpt, e.options)
			e.options = 0
		}
		if e.env != 0 {
			e.call(ortAPIReleaseEnv, e.env)
			e.env = 0
		}
		if e.handle != 0 {
			e.closeErr = purego.Dlclose(e.handle)
			e.handle = 0
		}
	})
	return e.closeErr
}

//go:uintptrescapes
func (e *Engine) call(offset uintptr, args ...uintptr) uintptr {
	function := *(*uintptr)(cPointer(e.api + offset))
	result, _, _ := purego.SyscallN(function, args...)
	return result
}

func (e *Engine) check(operation string, status uintptr) error {
	if status == 0 {
		return nil
	}
	message := "unknown ONNX Runtime error"
	if e.api != 0 {
		if pointer, _, _ := purego.SyscallN(*(*uintptr)(cPointer(e.api + ortAPIGetError)), status); pointer != 0 {
			message = cString(pointer)
		}
		purego.SyscallN(*(*uintptr)(cPointer(e.api + ortAPIReleaseStatus)), status)
	}
	return fmt.Errorf("inference: %s: %s", operation, message)
}

// cPointer reinterprets a C-owned address. Go cannot validate these pointers.
func cPointer(address uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&address))
}

func cString(pointer uintptr) string {
	if pointer == 0 {
		return ""
	}
	var result []byte
	for p := cPointer(pointer); len(result) < 4096; p = unsafe.Add(p, 1) {
		value := *(*byte)(p)
		if value == 0 {
			break
		}
		result = append(result, value)
	}
	return string(result)
}
