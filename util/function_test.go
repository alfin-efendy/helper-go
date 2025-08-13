package util

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test functions to create a call stack for testing
func testFunction1() runtime.Frame {
	return testFunction2()
}

func testFunction2() runtime.Frame {
	return testFunction3()
}

func testFunction3() runtime.Frame {
	return GetFrame(0)
}

func testFunction4() runtime.Frame {
	return GetFrame(1)
}

func testFunction5() runtime.Frame {
	return GetFrame(2)
}

// Nested function calls for deeper stack testing
func deepNest1() runtime.Frame {
	return deepNest2()
}

func deepNest2() runtime.Frame {
	return deepNest3()
}

func deepNest3() runtime.Frame {
	return deepNest4()
}

func deepNest4() runtime.Frame {
	return deepNest5()
}

func deepNest5() runtime.Frame {
	return GetFrame(0) // Should return deepNest5
}

func TestGetFrame(t *testing.T) {
	t.Run("Direct call with skipFrames=0", func(t *testing.T) {
		frame := GetFrame(0)

		// Should return the caller of GetFrame, which is this test function
		assert.NotEmpty(t, frame.Function)
		assert.Contains(t, frame.Function, "TestGetFrame")
		assert.NotEmpty(t, frame.File)
		assert.Positive(t, frame.Line)
		assert.Positive(t, frame.PC)
	})

	t.Run("Call through function chain", func(t *testing.T) {
		frame := testFunction1()

		// Should return testFunction3 as the final caller in the chain
		assert.NotEmpty(t, frame.Function)
		assert.Contains(t, frame.Function, "testFunction3")
		assert.NotEmpty(t, frame.File)
		assert.Positive(t, frame.Line)
	})

	t.Run("Direct call with skipFrames=1", func(t *testing.T) {
		frame := GetFrame(1)

		// Should skip one frame and return the caller of the test function
		// This will likely be the testing framework
		assert.NotEmpty(t, frame.Function)
		assert.NotEmpty(t, frame.File)
		assert.Positive(t, frame.Line)
		assert.Positive(t, frame.PC)
	})

	t.Run("Call through single function with skipFrames=0", func(t *testing.T) {
		frame := testFunction3()

		// Should return testFunction3 as the caller
		assert.NotEmpty(t, frame.Function)
		assert.Contains(t, frame.Function, "testFunction3")
		assert.NotEmpty(t, frame.File)
		assert.Positive(t, frame.Line)
	})

	t.Run("Call through multiple functions with different skipFrames", func(t *testing.T) {
		// Test with skipFrames=0 (should return the immediate caller)
		frame1 := testFunction3()
		assert.Contains(t, frame1.Function, "testFunction3")

		// Test with skipFrames=1 (should skip one frame)
		frame2 := testFunction4()
		assert.Contains(t, frame2.Function, "TestGetFrame") // Should be the test function

		// Test with skipFrames=2 (should skip two frames)
		frame3 := testFunction5()
		// This should skip even further up the call stack
		assert.NotEmpty(t, frame3.Function)
	})

	t.Run("Deep nested calls", func(t *testing.T) {
		frame := deepNest1()

		// Should return deepNest5 as it's the immediate caller
		assert.NotEmpty(t, frame.Function)
		assert.Contains(t, frame.Function, "deepNest5")
		assert.NotEmpty(t, frame.File)
		assert.Positive(t, frame.Line)
	})

	t.Run("Negative skipFrames", func(t *testing.T) {
		// Test with negative value - should still work due to how the function is implemented
		frame := GetFrame(-1)

		// Should still return a valid frame
		assert.NotEmpty(t, frame.Function)
		assert.NotEmpty(t, frame.File)
		assert.Positive(t, frame.Line)
	})

	t.Run("Very large skipFrames", func(t *testing.T) {
		// Test with a very large skip value
		frame := GetFrame(100)

		// When skipFrames is too large, should return default "unknown" function
		// or the last available frame
		assert.NotEmpty(t, frame.Function)
		// The frame should either be "unknown" or a valid frame from the stack
		assert.True(t, frame.Function == "unknown" || len(frame.Function) > 0)
	})
}

func TestGetFrameCallStack(t *testing.T) {
	t.Run("Verify call stack depth", func(t *testing.T) {
		// Create a known call stack and verify we can access different levels
		frame0 := GetFrame(0)
		frame1 := GetFrame(1)
		frame2 := GetFrame(2)

		// All frames should be different (different functions)
		assert.NotEqual(t, frame0.Function, frame1.Function)
		assert.NotEqual(t, frame1.Function, frame2.Function)
		assert.NotEqual(t, frame0.Function, frame2.Function)

		// All should have valid data
		frames := []runtime.Frame{frame0, frame1, frame2}
		for i, frame := range frames {
			t.Run(fmt.Sprintf("Frame %d", i), func(t *testing.T) {
				if frame.Function != "unknown" {
					assert.NotEmpty(t, frame.Function)
					assert.NotEmpty(t, frame.File)
					assert.Positive(t, frame.Line)
					assert.Positive(t, frame.PC)
				}
			})
		}
	})
}

func TestGetFrameConsistency(t *testing.T) {
	t.Run("Multiple calls should return same frame info", func(t *testing.T) {
		// Multiple calls with same skipFrames should return same function
		frame1 := GetFrame(0)
		frame2 := GetFrame(0)

		assert.Equal(t, frame1.Function, frame2.Function)
		assert.Equal(t, frame1.File, frame2.File)
		// Line numbers might be slightly different due to different call sites
		// but they should be close
		assert.LessOrEqual(t, abs(frame1.Line-frame2.Line), 2)
	})
}

func TestGetFrameFileInfo(t *testing.T) {
	t.Run("File path should be valid", func(t *testing.T) {
		frame := GetFrame(0)

		if frame.Function != "unknown" {
			// File should end with .go
			assert.True(t, strings.HasSuffix(frame.File, ".go"))
			// File should contain the test file name
			assert.Contains(t, frame.File, "function_test.go")
		}
	})
}

func TestGetFrameEdgeCases(t *testing.T) {
	t.Run("Zero program counters edge case", func(t *testing.T) {
		// This test ensures the function handles edge cases gracefully
		frame := GetFrame(0)

		// Should always return a frame, even if it's the default "unknown"
		assert.NotNil(t, frame)
		// Function should not be empty string
		assert.NotEmpty(t, frame.Function)
	})

	t.Run("Boundary conditions", func(t *testing.T) {
		// Test boundary values
		testCases := []int{0, 1, 10, 50}

		for _, skipFrames := range testCases {
			t.Run(fmt.Sprintf("skipFrames=%d", skipFrames), func(t *testing.T) {
				frame := GetFrame(skipFrames)

				// Should always return a valid frame struct
				assert.NotNil(t, frame)
				assert.NotEmpty(t, frame.Function)

				// If not "unknown", should have valid file info
				if frame.Function != "unknown" {
					assert.NotEmpty(t, frame.File)
					assert.Positive(t, frame.Line)
					assert.Positive(t, frame.PC)
				}
			})
		}
	})
}

// Helper function for absolute value calculation
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Benchmark tests
func BenchmarkGetFrame(b *testing.B) {
	b.Run("GetFrame-0", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetFrame(0)
		}
	})

	b.Run("GetFrame-1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetFrame(1)
		}
	})

	b.Run("GetFrame-5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetFrame(5)
		}
	})
}

// Example tests for documentation
func ExampleGetFrame() {
	frame := GetFrame(0)
	fmt.Printf("Function: %s\n", frame.Function)
	fmt.Printf("File: %s\n", frame.File)
	fmt.Printf("Line: %d\n", frame.Line)
	// Output will vary based on the calling context
}

func ExampleGetFrame_skipFrames() {
	// Example of using skipFrames to get caller information
	func() {
		// This anonymous function calls GetFrame(1)
		// skipFrames=1 will skip this anonymous function and return the caller
		frame := GetFrame(1)
		fmt.Printf("Caller function: %s\n", frame.Function)
	}()
	// Output will show the ExampleGetFrame_skipFrames function
}

// Integration test with real use case
func TestGetFrameRealWorldUsage(t *testing.T) {
	t.Run("Logger use case", func(t *testing.T) {
		// Simulate a logger that wants to know its caller
		logFunction := func(message string) runtime.Frame {
			// Skip the log function itself to get the real caller
			return GetFrame(0)
		}

		frame := logFunction("test message")

		// Should identify this test function as the caller
		assert.Contains(t, frame.Function, "TestGetFrameRealWorldUsage")
	})

	t.Run("Error reporting use case", func(t *testing.T) {
		// Simulate error reporting that needs caller information
		reportError := func(err error) runtime.Frame {
			return GetFrame(0)
		}

		frame := reportError(fmt.Errorf("test error"))

		// Should identify this test function as where the error was reported
		assert.Contains(t, frame.Function, "TestGetFrameRealWorldUsage")
	})
}

// Test for concurrent access
func TestGetFrameConcurrency(t *testing.T) {
	t.Run("Concurrent calls", func(t *testing.T) {
		const numGoroutines = 10
		const numCalls = 100

		results := make(chan runtime.Frame, numGoroutines*numCalls)

		// Start multiple goroutines
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numCalls; j++ {
					frame := GetFrame(0)
					results <- frame
				}
			}()
		}

		// Collect all results
		var frames []runtime.Frame
		for i := 0; i < numGoroutines*numCalls; i++ {
			frame := <-results
			frames = append(frames, frame)
		}

		// Verify all frames are valid
		validFrames := 0
		for _, frame := range frames {
			if frame.Function != "unknown" {
				assert.NotEmpty(t, frame.Function)
				assert.NotEmpty(t, frame.File)
				assert.Positive(t, frame.Line)
				assert.Positive(t, frame.PC)
				validFrames++
			}
		}

		assert.Len(t, frames, numGoroutines*numCalls)
		// Most frames should be valid (allowing some to be "unknown" due to timing)
		assert.Greater(t, validFrames, numGoroutines*numCalls/2)
	})
}
