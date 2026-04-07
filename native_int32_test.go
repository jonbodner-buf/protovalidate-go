// Copyright 2023-2026 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protovalidate

import (
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestNativeInt32Compare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rules  *validate.Int32Rules
		pass   []int32
		fail   []int32
		ruleID string
	}{
		// gt=5: pass {6}, fail {4, 5}
		{
			name:   "gt_only",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(5)}.Build(),
			pass:   []int32{6},
			fail:   []int32{4, 5},
			ruleID: "int32.gt",
		},
		// gte=5: pass {5, 6}, fail {4}
		{
			name:   "gte_only",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(5)}.Build(),
			pass:   []int32{5, 6},
			fail:   []int32{4},
			ruleID: "int32.gte",
		},
		// gt=0, lt=10: pass {1, 9}, fail {-1, 0, 10, 11}
		{
			name:   "gt_lt",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(0), Lt: proto.Int32(10)}.Build(),
			pass:   []int32{1, 9},
			fail:   []int32{-1, 0, 10, 11},
			ruleID: "int32.gt_lt",
		},
		// gt=10, lt=5 (exclusive): must be >10 or <5
		// pass {4, 11}, fail {5, 6, 9, 10}
		{
			name:   "gt_lt_exclusive",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(10), Lt: proto.Int32(5)}.Build(),
			pass:   []int32{4, 11},
			fail:   []int32{5, 6, 9, 10},
			ruleID: "int32.gt_lt_exclusive",
		},
		// gt=0, lte=10: pass {1, 9, 10}, fail {-1, 0, 11}
		{
			name:   "gt_lte",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(0), Lte: proto.Int32(10)}.Build(),
			pass:   []int32{1, 9, 10},
			fail:   []int32{-1, 0, 11},
			ruleID: "int32.gt_lte",
		},
		// gt=10, lte=5 (exclusive): must be >10 or <=5
		// pass {4, 5, 11}, fail {6, 9, 10}
		{
			name:   "gt_lte_exclusive",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(10), Lte: proto.Int32(5)}.Build(),
			pass:   []int32{4, 5, 11},
			fail:   []int32{6, 9, 10},
			ruleID: "int32.gt_lte_exclusive",
		},
		// gte=0, lt=10: pass {0, 1, 9}, fail {-1, 10, 11}
		{
			name:   "gte_lt",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(0), Lt: proto.Int32(10)}.Build(),
			pass:   []int32{0, 1, 9},
			fail:   []int32{-1, 10, 11},
			ruleID: "int32.gte_lt",
		},
		// gte=10, lt=5 (exclusive): must be >=10 or <5
		// pass {4, 10, 11}, fail {5, 6, 9}
		{
			name:   "gte_lt_exclusive",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(10), Lt: proto.Int32(5)}.Build(),
			pass:   []int32{4, 10, 11},
			fail:   []int32{5, 6, 9},
			ruleID: "int32.gte_lt_exclusive",
		},
		// gte=0, lte=10: pass {0, 1, 9, 10}, fail {-1, 11}
		{
			name:   "gte_lte",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(0), Lte: proto.Int32(10)}.Build(),
			pass:   []int32{0, 1, 9, 10},
			fail:   []int32{-1, 11},
			ruleID: "int32.gte_lte",
		},
		// gte=10, lte=5 (exclusive): must be >=10 or <=5
		// pass {4, 5, 10, 11}, fail {6, 9}
		{
			name:   "gte_lte_exclusive",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(10), Lte: proto.Int32(5)}.Build(),
			pass:   []int32{4, 5, 10, 11},
			fail:   []int32{6, 9},
			ruleID: "int32.gte_lte_exclusive",
		},
		// Equal bound edge cases: empty ranges that always fail.
		{
			name:   "gte_eq_lt",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(5), Lt: proto.Int32(5)}.Build(),
			fail:   []int32{4, 5, 6},
			ruleID: "int32.gte_lt",
		},
		{
			name:   "gt_eq_lt",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(5), Lt: proto.Int32(5)}.Build(),
			fail:   []int32{4, 5, 6},
			ruleID: "int32.gt_lt",
		},
		{
			name:   "gt_eq_lte",
			rules:  validate.Int32Rules_builder{Gt: proto.Int32(5), Lte: proto.Int32(5)}.Build(),
			fail:   []int32{4, 5, 6},
			ruleID: "int32.gt_lte",
		},
		// Equal bound: gte=5, lte=5 → only 5 passes.
		{
			name:   "gte_eq_lte",
			rules:  validate.Int32Rules_builder{Gte: proto.Int32(5), Lte: proto.Int32(5)}.Build(),
			pass:   []int32{5},
			fail:   []int32{4, 6},
			ruleID: "int32.gte_lte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			eval := buildNativeInt32(t, tt.rules)
			require.NotNil(t, eval)
			for _, v := range tt.pass {
				err := eval.Evaluate(nil, protoreflect.ValueOfInt32(v), &validationConfig{})
				assert.NoError(t, err, "expected %d to pass", v)
			}
			for _, v := range tt.fail {
				err := eval.Evaluate(nil, protoreflect.ValueOfInt32(v), &validationConfig{})
				require.Error(t, err, "expected %d to fail", v)
				var valErr *ValidationError
				require.ErrorAs(t, err, &valErr)
				require.Len(t, valErr.Violations, 1)
				assert.Equal(t, tt.ruleID, valErr.Violations[0].Proto.GetRuleId())
			}
		})
	}
}

func TestNativeInt32Compare_FieldValue(t *testing.T) {
	t.Parallel()
	eval := buildNativeInt32(t, validate.Int32Rules_builder{Gt: proto.Int32(5)}.Build())
	require.NotNil(t, eval)
	val := protoreflect.ValueOfInt32(3)
	err := eval.Evaluate(nil, val, &validationConfig{})
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Len(t, valErr.Violations, 1)
	assert.Equal(t, int64(3), valErr.Violations[0].FieldValue.Int())
}

func TestNativeInt32Compare_Tautology(t *testing.T) {
	t.Parallel()
	eval := buildNativeInt32(t, validate.Int32Rules_builder{Gt: proto.Int32(0)}.Build())
	require.NotNil(t, eval)
	assert.False(t, eval.Tautology())
}

func TestTryBuildNativeInt32Rules_ReturnsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rules *validate.Int32Rules
	}{
		{"nil_rules", nil},
		{"const", validate.Int32Rules_builder{Const: proto.Int32(5)}.Build()},
		{"in", validate.Int32Rules_builder{In: []int32{1, 2, 3}}.Build()},
		{"not_in", validate.Int32Rules_builder{NotIn: []int32{1, 2, 3}}.Build()},
		{"lt_only", validate.Int32Rules_builder{Lt: proto.Int32(10)}.Build()},
		{"lte_only", validate.Int32Rules_builder{Lte: proto.Int32(10)}.Build()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, tryBuildNativeInt32Rules(base{}, tt.rules))
		})
	}
}

// buildNativeInt32 constructs a nativeInt32Compare evaluator for testing.
func buildNativeInt32(t *testing.T, rules *validate.Int32Rules) evaluator {
	t.Helper()
	fdesc := newInt32FieldDescriptor(t)
	b := base{
		Descriptor:       fdesc,
		FieldPathElement: fieldPathElement(fdesc),
	}
	return tryBuildNativeInt32Rules(b, rules)
}

// newInt32FieldDescriptor creates a minimal int32 field descriptor for testing.
func newInt32FieldDescriptor(t *testing.T) protoreflect.FieldDescriptor {
	t.Helper()
	fileProto := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Msg"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("val"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
		},
		Syntax: proto.String("proto3"),
	}
	file, err := protodesc.NewFile(fileProto, nil)
	require.NoError(t, err)
	return file.Messages().Get(0).Fields().Get(0)
}

// TestNativeInt32_EndToEnd validates that the native evaluator is used
// for int32 fields via the full validator pipeline with dynamic messages.
func TestNativeInt32_EndToEnd(t *testing.T) {
	t.Parallel()

	msgType := newDynamicMessageType(t, "test.native", "IntMsg", &descriptorpb.FieldDescriptorProto{
		Name:   proto.String("value"),
		Number: proto.Int32(1),
		Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Options: fieldOpts(validate.FieldRules_builder{
			Int32: validate.Int32Rules_builder{Gt: proto.Int32(0)}.Build(),
		}.Build()),
	})

	validator, err := New(WithDisableLazy(), WithMessageDescriptors(msgType.Descriptor()))
	require.NoError(t, err)

	passing := dynamicpb.NewMessage(msgType.Descriptor())
	passing.Set(msgType.Descriptor().Fields().ByName("value"), protoreflect.ValueOfInt32(1))
	assert.NoError(t, validator.Validate(passing))

	failing := dynamicpb.NewMessage(msgType.Descriptor())
	failing.Set(msgType.Descriptor().Fields().ByName("value"), protoreflect.ValueOfInt32(0))
	err = validator.Validate(failing)
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Len(t, valErr.Violations, 1)
	assert.Equal(t, "int32.gt", valErr.Violations[0].Proto.GetRuleId())
	assert.Equal(t, "value must be greater than 0", valErr.Violations[0].Proto.GetMessage())
}
