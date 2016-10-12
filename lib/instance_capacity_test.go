package autoscaler

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCapacityFromInstanceType(t *testing.T) {
	_, err := CapacityFromInstanceType("unknown")
	assert.Error(t, err)
}
