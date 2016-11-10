package shapes_test

import (
	"dynagrok/examples/shapes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvironment(t *testing.T) {
	assert.True(t, true)
}

func TestWindow(t *testing.T) {
	w := shapes.InitWindow(400, 600)
	assert.Equal(t, 400, w.Height(), "Height unsuccessfully initialized")
	assert.Equal(t, 600, w.Width(), "Width unsuccessfully initialized")
}
