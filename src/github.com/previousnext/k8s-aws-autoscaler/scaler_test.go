package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScaler(t *testing.T) {
	assert.Equal(t, 5, scaler(2580, 17518, InstanceType{
		Name:   "t2.medium",
		CPU:    2000,
		Memory: 4000,
	}), "Scale set by Memory")

	assert.Equal(t, 9, scaler(17518, 2580, InstanceType{
		Name:   "t2.medium",
		CPU:    2000,
		Memory: 4000,
	}), "Scale set by CPU")

	assert.Equal(t, 1, scaler(20, 200, InstanceType{
		Name:   "t2.medium",
		CPU:    2000,
		Memory: 4000,
	}), "Tiny cluster has single node")
}
