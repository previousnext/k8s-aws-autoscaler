package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstanceType(t *testing.T) {
	i, err := getInstanceType("t2.medium")
	assert.Nil(t, err)

	assert.Equal(t, InstanceType{Name: "t2.medium", CPU: 2000, Memory: 4000}, i, "Found t2.medium")

	i, err = getInstanceType("t20.medium")
	assert.NotNil(t, err)
}
