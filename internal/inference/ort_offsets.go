package inference

// These offsets are generated from onnxruntime_c_api.h for ONNX Runtime 1.23.2,
// ORT_API_VERSION 23, on 64-bit platforms. Re-audit them before changing the
// pinned runtime.
const (
	ortAPIVersion = 23

	ortBaseGetAPI       = 0
	ortBaseGetVersion   = 8
	ortAPIGetError      = 16
	ortAPICreateEnv     = 24
	ortAPICreateSession = 64
	ortAPIRun           = 72
	ortAPICreateOptions = 80

	ortAPISetGraphOptimization = 184
	ortAPISetIntraOpThreads    = 192

	ortAPICreateTensor     = 392
	ortAPIGetTensorData    = 408
	ortAPIGetDimensions    = 496
	ortAPIGetDimensionNum  = 488
	ortAPIGetTypeAndShape  = 520
	ortAPICreateCPUMemInfo = 552

	ortAPIReleaseEnv        = 736
	ortAPIReleaseStatus     = 744
	ortAPIReleaseMemInfo    = 752
	ortAPIReleaseSession    = 760
	ortAPIReleaseValue      = 768
	ortAPIReleaseTypeShape  = 792
	ortAPIReleaseSessionOpt = 800
)
