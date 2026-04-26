package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjustOffset(t *testing.T) {
	visible := 48 - defaultReservedRows - 1
	tests := []struct {
		name     string
		cursor   int
		offset   int
		expected int
	}{
		{name: "cursor went below visible", cursor: visible, offset: 0, expected: 1},
		{name: "cursor went above visible", cursor: 0, offset: 1, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModelEmpty()

			m.cursor = tt.cursor
			m.offset = tt.offset
			m.adjustOffset()

			assert.Equal(t, tt.expected, m.offset)
		})
	}
}

func TestIsAncestor(t *testing.T) {
	assert.True(t, isAncestor("module.a", "module.a.module.b.aws_s3.x"))
	assert.False(t, isAncestor("module.b", "module.a.module.b.aws_s3.x"))
	assert.False(t, isAncestor("module.a", "aws_s3.x"))
}

func TestParentModule(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{name: "normal resource", address: "module.a.aws_s3.x", expected: "module.a"},
		{name: "normal module", address: "module.a.module.b", expected: "module.a"},
		{name: "resource no parent", address: "aws_s3.x", expected: ""},
		{name: "module no parent", address: "module.a", expected: ""},
		{name: "resource with module bracket and dot", address: "module.vpc[\"a.b\"].aws_s3.x", expected: "module.vpc[\"a.b\"]"},
		{name: "data under module", address: "module.api.data.aws_region.current", expected: "module.api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parentModule(tt.address))
		})
	}
}
