package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstanceType(t *testing.T) {
	i, _ := getInstanceType("t2.medium")
	assert.Equal(t, InstanceType{
		Name:   "t2.medium",
		CPU:    2000,
		Memory: 4000,
	}, i, "Found t2.medium")

	i, _ = getInstanceType("t2.medium")
	assert.Equal(t, InstanceType{}, i, "Could not find t20.medium")
}
