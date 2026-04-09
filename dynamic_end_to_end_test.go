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

func TestDynamicRulesEndToEnd(t *testing.T) {
	data := []struct {
		name string
		typ  *descriptorpb.FieldDescriptorProto_Type
		rule *validate.FieldRules
		info dynamicMessageTesterInfo
	}{
		{
			name: "int32_gt",
			typ:  descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
			rule: validate.FieldRules_builder{
				Int32: validate.Int32Rules_builder{Gt: proto.Int32(0)}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfInt32(1),
				badValue:          protoreflect.ValueOfInt32(0),
				failedRuleID:      "int32.gt",
				failedRuleMessage: "value must be greater than 0",
			},
		},
		{
			name: "uint64_gte",
			typ:  descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
			rule: validate.FieldRules_builder{
				Uint64: validate.UInt64Rules_builder{Gte: proto.Uint64(10)}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfUint64(10),
				badValue:          protoreflect.ValueOfUint64(9),
				failedRuleID:      "uint64.gte",
				failedRuleMessage: "value must be greater than or equal to 10",
			},
		},
		{
			name: "double_lt",
			typ:  descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(),
			rule: validate.FieldRules_builder{
				Double: validate.DoubleRules_builder{Lt: proto.Float64(100)}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfFloat64(50),
				badValue:          protoreflect.ValueOfFloat64(100),
				failedRuleID:      "double.lt",
				failedRuleMessage: "value must be less than 100",
			},
		},
		{
			name: "min_len",
			typ:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			rule: validate.FieldRules_builder{
				String: validate.StringRules_builder{MinLen: proto.Uint64(3)}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfString("abc"),
				badValue:          protoreflect.ValueOfString("ab"),
				failedRuleID:      "string.min_len",
				failedRuleMessage: "value length must be at least 3 characters",
			},
		},
		{
			name: "prefix",
			typ:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			rule: validate.FieldRules_builder{
				String: validate.StringRules_builder{Prefix: proto.String("hello")}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfString("hello world"),
				badValue:          protoreflect.ValueOfString("world"),
				failedRuleID:      "string.prefix",
				failedRuleMessage: "value does not have prefix `hello`",
			},
		},
		{
			name: "bool_const",
			typ:  descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
			rule: validate.FieldRules_builder{
				Bool: validate.BoolRules_builder{Const: proto.Bool(true)}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfBool(true),
				badValue:          protoreflect.ValueOfBool(false),
				failedRuleID:      "bool.const",
				failedRuleMessage: "value must equal true",
			},
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			// we are setting an environment variable to enable native rules, so we can't use parallel tests
			msgType := newDynamicMessageType(t, "test.native", "TestMessage", &descriptorpb.FieldDescriptorProto{
				Name:    proto.String("value"),
				Number:  proto.Int32(1),
				Type:    d.typ,
				Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Options: fieldOpts(d.rule),
			})
			d.info.msgType = msgType
			// first with CEL rules
			t.Setenv("PV_NATIVE_RULES", "false")
			dynamicMessageTester(t, d.info)
			// now with native rules to validate they produce identical results
			t.Setenv("PV_NATIVE_RULES", "true")
			dynamicMessageTester(t, d.info)
		})
	}
}

func TestNativeEnum_EndToEnd(t *testing.T) {
	// Build a proto with an enum field and const rule.
	enumDesc := &descriptorpb.EnumDescriptorProto{
		Name: proto.String("TestEnum"),
		Value: []*descriptorpb.EnumValueDescriptorProto{
			{Name: proto.String("UNSPECIFIED"), Number: proto.Int32(0)},
			{Name: proto.String("VALUE_ONE"), Number: proto.Int32(1)},
			{Name: proto.String("VALUE_TWO"), Number: proto.Int32(2)},
		},
	}
	data := []struct {
		name string
		rule *validate.FieldRules
		info dynamicMessageTesterInfo
	}{
		{
			name: "enum_const",
			rule: validate.FieldRules_builder{
				Enum: validate.EnumRules_builder{Const: proto.Int32(1)}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfEnum(1),
				badValue:          protoreflect.ValueOfEnum(2),
				failedRuleID:      "enum.const",
				failedRuleMessage: "value must equal 1",
			},
		},
		{
			name: "enum_in",
			rule: validate.FieldRules_builder{
				Enum: validate.EnumRules_builder{In: []int32{1, 2}}.Build(),
			}.Build(),
			info: dynamicMessageTesterInfo{
				goodValue:         protoreflect.ValueOfEnum(1),
				badValue:          protoreflect.ValueOfEnum(3),
				failedRuleID:      "enum.in",
				failedRuleMessage: "value must be in list [1, 2]",
			},
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			msgType := newDynamicMessageTypeWithEnum(t, "test.native", "EnumMsg", enumDesc, &descriptorpb.FieldDescriptorProto{
				Name:     proto.String("value"),
				Number:   proto.Int32(1),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
				TypeName: proto.String(".test.native.TestEnum"),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Options:  fieldOpts(d.rule),
			})
			d.info.msgType = msgType
			// first with CEL rules
			t.Setenv("PV_NATIVE_RULES", "false")
			dynamicMessageTester(t, d.info)
			// now with native rules to validate they produce identical results
			t.Setenv("PV_NATIVE_RULES", "true")
			dynamicMessageTester(t, d.info)
		})
	}
}

type dynamicMessageTesterInfo struct {
	msgType           protoreflect.MessageType
	goodValue         protoreflect.Value
	badValue          protoreflect.Value
	failedRuleID      string
	failedRuleMessage string
}

func dynamicMessageTester(t *testing.T, info dynamicMessageTesterInfo) {
	t.Helper()
	validator, err := New(WithDisableLazy(), WithMessageDescriptors(info.msgType.Descriptor()))
	require.NoError(t, err)

	passing := dynamicpb.NewMessage(info.msgType.Descriptor())
	passing.Set(info.msgType.Descriptor().Fields().ByName("value"), info.goodValue)
	require.NoError(t, validator.Validate(passing))

	failing := dynamicpb.NewMessage(info.msgType.Descriptor())
	failing.Set(info.msgType.Descriptor().Fields().ByName("value"), info.badValue)
	err = validator.Validate(failing)
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	require.Len(t, valErr.Violations, 1)
	assert.Equal(t, info.failedRuleID, valErr.Violations[0].Proto.GetRuleId())
	assert.Equal(t, info.failedRuleMessage, valErr.Violations[0].Proto.GetMessage())
}

// newDynamicMessageTypeWithEnum creates a dynamic message type that includes
// an enum type definition.
func newDynamicMessageTypeWithEnum(
	t testing.TB,
	pkg, name string,
	enumDesc *descriptorpb.EnumDescriptorProto,
	field *descriptorpb.FieldDescriptorProto,
) protoreflect.MessageType {
	t.Helper()

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String(pkg + "." + name + ".proto"),
		Package: proto.String(pkg),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"buf/validate/validate.proto",
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{enumDesc},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:  proto.String(name),
			Field: []*descriptorpb.FieldDescriptorProto{field},
		}},
	}

	registry := newRegistryWithValidateProto(t)
	fd, err := protodesc.FileOptions{}.New(file, registry)
	require.NoError(t, err)

	desc := fd.Messages().ByName(protoreflect.Name(name))
	require.NotNil(t, desc)

	return dynamicpb.NewMessageType(desc)
}
