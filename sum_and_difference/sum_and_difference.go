// This is a basic command-line application that serves as a minimal example
// for loading an executing a neural network using the onnxruntime library.
//
// If you're wanting to learn how to use the onnxruntime_go library, the
// runTest function is the most important one here.  The rest of this program
// is mostly boilerplate for setting up a command-line program.
//
// The actual network used by this program was generated by the included
// generate_network.py pytorch script. It takes a 1x4 input vector of 32-bit
// floats, and produces a 1x2 output vector of 32-bit floats. The network
// attempts to populate the two values in the output vector with 1) the sum
// of the four inputs, and 2), the maximum difference between any two of the
// input values.
package main

import (
	"flag"
	"fmt"
	ort "github.com/yalue/onnxruntime_go"
	"os"
	"runtime"
)

// Attempts to find and return a path to a version of the onnxruntime shared
// library compatible with the current OS and system architecture.
func getDefaultSharedLibPath() string {
	// For now, we only include libraries for x86_64 windows, ARM64 darwin, and
	// x86_64 or ARM64 Linux. In the future, libraries may be added or removed.
	if runtime.GOOS == "windows" {
		if runtime.GOARCH == "amd64" {
			return "../third_party/onnxruntime.dll"
		}
	}
	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			return "../third_party/onnxruntime_arm64.dylib"
		}
	}
	if runtime.GOOS == "linux" {
		if runtime.GOARCH == "arm64" {
			return "../third_party/onnxruntime_arm64.so"
		}
		return "../third_party/onnxruntime.so"
	}
	fmt.Printf("Unable to determine a path to the onnxruntime shared library"+
		" for OS \"%s\" and architecture \"%s\".\n", runtime.GOOS,
		runtime.GOARCH)
	return ""
}

// Actually sets up and runs the neural network. Requires a path to the
// onnxruntime shared library file.
func runTest(onnxruntimeLibPath string) error {
	// Step 1: Initialize the onnxruntime library after providing a path to the
	// shared library to use.
	ort.SetSharedLibraryPath(onnxruntimeLibPath)
	e := ort.InitializeEnvironment()
	if e != nil {
		return fmt.Errorf("Error initializing the onnxruntime library: %w", e)
	}
	// Clean up the onnxruntime library when we're done using it.
	defer ort.DestroyEnvironment()

	// Step 2: Create the input tensor. Tensors are wrappers around Go slices;
	// the onnxruntime networks access the data in these slices to read inputs
	// or write outputs. Here, we'll create a 1x4 input tensor initialized
	// with some preset data. The inputData slice can be modified directly to
	// change the input values, even after creating the inputTensor.
	inputData := []float32{0.2, 0.3, 0.6, 0.9}
	// The tensor's shape is actually 1x1x4 rather than 1x4 because the first
	// dimension in the PyTorch script was used for batch size.
	inputTensor, e := ort.NewTensor(ort.NewShape(1, 1, 4), inputData)
	if e != nil {
		return fmt.Errorf("Error creating the input tensor: %w", e)
	}
	// Tensors must always be destroyed when they're no longer needed to free
	// associated onnxruntime structures. Destroying the tensor object won't
	// change the underlying Go data slice, which can still be cleaned up by
	// Go's garbage collector when it's no longer referenced.
	defer inputTensor.Destroy()

	// Step 3: Create the output tensor. Since we don't need to initialize it,
	// we can use the NewEmptyTensor to just get a zero-filled tensor with the
	// required shape for this network. The library will automatically allocate
	// a Go slice with the necessary capacity in this case. To access this
	// slice, we can call outputTensor.GetData() after creating the tensor.
	outputTensor, e := ort.NewEmptyTensor[float32](ort.NewShape(1, 1, 2))
	if e != nil {
		return fmt.Errorf("Error creating the output tensor: %w", e)
	}
	defer outputTensor.Destroy()

	// Step 4: Load the network itself into an onnxruntime Session instance.
	// Note that we call "NewAdvancedSession"---this isn't particularly
	// "Advanced", but it's simply a newer version of the API that allows
	// specifying additional options (which we don't use here). onnxruntime
	// requires associating input and output tensors with names, which in this
	// case we set to "1x4 Input Vector" and "1x2 Output Vector" when creating
	// the network. (If you're curious, this was done when exporting the .onnx
	// file from the the python script.) The last argument to
	// NewAdvancedSession is a pointer to a SessionOptions instance, which we
	// leave as nil to indicate that default options are OK.
	session, e := ort.NewAdvancedSession("./sum_and_difference.onnx",
		[]string{"1x4 Input Vector"},
		[]string{"1x2 Output Vector"},
		[]ort.ArbitraryTensor{inputTensor},
		[]ort.ArbitraryTensor{outputTensor},
		nil)
	if e != nil {
		return fmt.Errorf("Error creating the session: %w", e)
	}
	// The session must also always be destroyed to free internal data.
	// Destroying the session will not modify or destroy the input or output
	// tensors it was using.
	defer session.Destroy()

	// Step 5: Actually run the network. This will read the data from the input
	// tensor, and write to the output tensor. To re-run the network with
	// different inputs, we can simply modify the inputData slice before
	// calling Run() again. (Here, we only call it once, though.)
	e = session.Run()
	if e != nil {
		return fmt.Errorf("Error executing the network: %w", e)
	}

	// Step 6: Read the output data and present the results. The network may
	// not be very good, but it was designed to be a small test and not trained
	// for very long!
	outputData := outputTensor.GetData()
	fmt.Printf("The network ran without errors.\n")
	fmt.Printf("  Input data: %v\n", inputData)
	fmt.Printf("  Approximate sum of inputs: %f\n", outputData[0])
	fmt.Printf("  Approximate max difference between any two inputs: %f\n", outputData[1])
	return nil
}

func run() int {
	var onnxruntimeLibPath string
	flag.StringVar(&onnxruntimeLibPath, "onnxruntime_lib",
		getDefaultSharedLibPath(),
		"The path to the onnxruntime shared library for your system.")
	flag.Parse()
	if onnxruntimeLibPath == "" {
		fmt.Println("You must specify a path to the onnxruntime shared " +
			"on your system. Run with -help for more information.")
		return 1
	}
	e := runTest(onnxruntimeLibPath)
	if e != nil {
		fmt.Printf("Encountered an error running the network: %s\n", e)
		return 1
	}
	fmt.Printf("The network seemed to run OK!\n")
	return 0
}

func main() {
	os.Exit(run())
}
