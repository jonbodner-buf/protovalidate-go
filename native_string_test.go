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
)

func buildNativeString(t testing.TB, rules *validate.StringRules) evaluator {
	t.Helper()
	fdesc := newFieldDescriptor(t, descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum())
	b := base{
		Descriptor:       fdesc,
		FieldPathElement: fieldPathElement(fdesc),
	}
	return tryBuildNativeStringRules(b, rules)
}

func TestNativeStringConst(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Const: proto.String("hello")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("hello"), &validationConfig{}))

	err := eval.Evaluate(nil, protoreflect.ValueOfString("world"), &validationConfig{})
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Len(t, valErr.Violations, 1)
	assert.Equal(t, "string.const", valErr.Violations[0].Proto.GetRuleId())
	assert.Equal(t, "value must equal `hello`", valErr.Violations[0].Proto.GetMessage())
}

func TestNativeStringLen(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Len: proto.Uint64(3)}.Build())
	require.NotNil(t, eval)

	// ASCII
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc"), &validationConfig{}))
	// Unicode: 3 code points
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("αβγ"), &validationConfig{}))

	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("ab"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("abcd"), &validationConfig{}))

	err := eval.Evaluate(nil, protoreflect.ValueOfString("ab"), &validationConfig{})
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "string.len", valErr.Violations[0].Proto.GetRuleId())
}

func TestNativeStringMinLen(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{MinLen: proto.Uint64(2)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("ab"), &validationConfig{}))
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("a"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString(""), &validationConfig{}))
}

func TestNativeStringMaxLen(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{MaxLen: proto.Uint64(3)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString(""), &validationConfig{}))
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("abcd"), &validationConfig{}))
}

func TestNativeStringLenBytes(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{LenBytes: proto.Uint64(4)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("abcd"), &validationConfig{}))
	// "αβ" is 4 bytes (2 bytes per Greek letter)
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("αβ"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("abcde"), &validationConfig{}))
}

func TestNativeStringMinBytes(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{MinBytes: proto.Uint64(2)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("ab"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("a"), &validationConfig{}))
}

func TestNativeStringMaxBytes(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{MaxBytes: proto.Uint64(3)}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("abcd"), &validationConfig{}))
}

func TestNativeStringPattern(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Pattern: proto.String("^[a-z]+$")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("ABC"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("abc1"), &validationConfig{}))

	err := eval.Evaluate(nil, protoreflect.ValueOfString("ABC"), &validationConfig{})
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "string.pattern", valErr.Violations[0].Proto.GetRuleId())
	assert.Equal(t, "value does not match regex pattern `^[a-z]+$`", valErr.Violations[0].Proto.GetMessage())
}

func TestNativeStringPattern_InvalidRegex(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Pattern: proto.String("[invalid")}.Build())
	assert.Nil(t, eval, "invalid regex should bail to CEL")
}

func TestNativeStringPrefix(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Prefix: proto.String("foo")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("foobar"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("barfoo"), &validationConfig{}))
}

func TestNativeStringSuffix(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Suffix: proto.String("bar")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("foobar"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("barbaz"), &validationConfig{}))
}

func TestNativeStringContains(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{Contains: proto.String("mid")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("amidst"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("absent"), &validationConfig{}))
}

func TestNativeStringNotContains(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{NotContains: proto.String("bad")}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("good"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("badger"), &validationConfig{}))
}

func TestNativeStringIn(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{In: []string{"a", "b", "c"}}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("a"), &validationConfig{}))
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("b"), &validationConfig{}))

	err := eval.Evaluate(nil, protoreflect.ValueOfString("d"), &validationConfig{})
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "string.in", valErr.Violations[0].Proto.GetRuleId())
}

func TestNativeStringNotIn(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{NotIn: []string{"x", "y"}}.Build())
	require.NotNil(t, eval)

	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("a"), &validationConfig{}))
	require.Error(t, eval.Evaluate(nil, protoreflect.ValueOfString("x"), &validationConfig{}))
}

func TestNativeStringUnicode(t *testing.T) {
	t.Parallel()

	// min_len counts runes, not bytes
	eval := buildNativeString(t, validate.StringRules_builder{MinLen: proto.Uint64(1)}.Build())
	require.NotNil(t, eval)
	require.NoError(t, eval.Evaluate(nil, protoreflect.ValueOfString("α"), &validationConfig{})) // 1 rune, 2 bytes

	// max_bytes counts bytes
	eval2 := buildNativeString(t, validate.StringRules_builder{MaxBytes: proto.Uint64(1)}.Build())
	require.NotNil(t, eval2)
	require.Error(t, eval2.Evaluate(nil, protoreflect.ValueOfString("α"), &validationConfig{})) // 2 bytes > 1
}

// --- Bail-out tests ---

func TestTryBuildNativeStringRules_ReturnsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rules *validate.StringRules
	}{
		{"nil_rules", nil},
		{"empty_rules", validate.StringRules_builder{}.Build()},
		{"email", validate.StringRules_builder{Email: proto.Bool(true)}.Build()},
		{"hostname", validate.StringRules_builder{Hostname: proto.Bool(true)}.Build()},
		{"uuid", validate.StringRules_builder{Uuid: proto.Bool(true)}.Build()},
		{"ip", validate.StringRules_builder{Ip: proto.Bool(true)}.Build()},
		{"uri", validate.StringRules_builder{Uri: proto.Bool(true)}.Build()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, tryBuildNativeStringRules(base{}, tt.rules))
		})
	}
}

func TestNativeStringTautology(t *testing.T) {
	t.Parallel()
	eval := buildNativeString(t, validate.StringRules_builder{MinLen: proto.Uint64(1)}.Build())
	require.NotNil(t, eval)
	assert.False(t, eval.Tautology())
}
