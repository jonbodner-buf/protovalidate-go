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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestNativeBytesConst(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{Const: []byte{0x01, 0x02}}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01, 0x02}), &validationConfig{}))

	err := eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x03}), &validationConfig{})
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Len(t, valErr.Violations, 1)
	assert.Equal(t, "bytes.const", valErr.Violations[0].Proto.GetRuleId())
	assert.Equal(t, "value must be 0102", valErr.Violations[0].Proto.GetMessage())
}

func TestNativeBytesLen(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{Len: proto.Uint64(4)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{1, 2, 3, 4}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{1, 2, 3}), &validationConfig{}))
}

func TestNativeBytesMinLen(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{MinLen: proto.Uint64(2)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{1, 2}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{1}), &validationConfig{}))
}

func TestNativeBytesMaxLen(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{MaxLen: proto.Uint64(3)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{1, 2, 3}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{1, 2, 3, 4}), &validationConfig{}))
}

func TestNativeBytesPattern(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{Pattern: proto.String("^[a-zA-Z0-9]+$")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte("abc123")), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte("abc 123")), &validationConfig{}))
}

func TestNativeBytesPrefix(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{Prefix: []byte{0x01, 0x02}}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01, 0x02, 0x03}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x03, 0x02, 0x01}), &validationConfig{}))
}

func TestNativeBytesSuffix(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{Suffix: []byte{0x03}}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01, 0x02, 0x03}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x03, 0x02, 0x01}), &validationConfig{}))
}

func TestNativeBytesContains(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{Contains: []byte{0x02}}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01, 0x02, 0x03}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01, 0x03}), &validationConfig{}))
}

func TestNativeBytesIn(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{
		In: [][]byte{{0x01}, {0x02}},
	}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x03}), &validationConfig{}))
}

func TestNativeBytesNotIn(t *testing.T) {
	t.Parallel()
	eval := buildNativeBytes(t, validate.BytesRules_builder{
		NotIn: [][]byte{{0x00}},
	}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x01}), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfBytes([]byte{0x00}), &validationConfig{}))
}

func TestTryBuildNativeBytesRules_ReturnsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rules *validate.BytesRules
	}{
		{"nil_rules", nil},
		{"empty_rules", validate.BytesRules_builder{}.Build()},
		{"ip_well_known", validate.BytesRules_builder{Ip: proto.Bool(true)}.Build()},
		{"ipv4_well_known", validate.BytesRules_builder{Ipv4: proto.Bool(true)}.Build()},
		{"ipv6_well_known", validate.BytesRules_builder{Ipv6: proto.Bool(true)}.Build()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, tryBuildNativeBytesRules(base{}, tt.rules))
		})
	}
}

func TestNativeBytes_EndToEnd(t *testing.T) {
	t.Setenv("PV_NATIVE_RULES", "true")

	msgType := newDynamicMessageType(t, "test.native", "BytesMsg", &descriptorpb.FieldDescriptorProto{
		Name:   proto.String("value"),
		Number: proto.Int32(1),
		Type:   descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Options: fieldOpts(validate.FieldRules_builder{
			Bytes: validate.BytesRules_builder{MinLen: proto.Uint64(2)}.Build(),
		}.Build()),
	})

	validator, err := New(WithDisableLazy(), WithMessageDescriptors(msgType.Descriptor()))
	require.NoError(t, err)

	passing := dynamicpb.NewMessage(msgType.Descriptor())
	passing.Set(msgType.Descriptor().Fields().ByName("value"), protoreflect.ValueOfBytes([]byte{0x01, 0x02}))
	require.NoError(t, validator.Validate(passing))

	failing := dynamicpb.NewMessage(msgType.Descriptor())
	failing.Set(msgType.Descriptor().Fields().ByName("value"), protoreflect.ValueOfBytes([]byte{0x01}))
	err = validator.Validate(failing)
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Len(t, valErr.Violations, 1)
	assert.Equal(t, "bytes.min_len", valErr.Violations[0].Proto.GetRuleId())
	assert.Equal(t, "value length must be at least 2 bytes", valErr.Violations[0].Proto.GetMessage())
}

func buildNativeBytes(t testing.TB, rules *validate.BytesRules) evaluator {
	t.Helper()
	fdesc := newFieldDescriptor(t, descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum())
	b := base{
		Descriptor:       fdesc,
		FieldPathElement: fieldPathElement(fdesc),
	}
	return tryBuildNativeBytesRules(b, rules)
}
