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
	"fmt"
	"slices"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//nolint:gochecknoglobals
var (
	enumConstDesc = (*validate.EnumRules)(nil).ProtoReflect().Descriptor().Fields().ByName("const")
	enumInDesc    = (*validate.EnumRules)(nil).ProtoReflect().Descriptor().Fields().ByName("in")
	enumNotInDesc = (*validate.EnumRules)(nil).ProtoReflect().Descriptor().Fields().ByName("not_in")
)

// nativeEnumEval is a native Go evaluator for enum const/in/not_in rules.
type nativeEnumEval struct {
	base
	constVal  *int32
	inVals    []int32
	notInVals []int32
}

func (n nativeEnumEval) Evaluate(_ protoreflect.Message, val protoreflect.Value, _ *validationConfig) error {
	enumVal := int32(val.Enum())

	// const
	if n.constVal != nil && enumVal != *n.constVal {
		return n.violationError(
			"enum.const",
			fmt.Sprintf("value must equal %d", *n.constVal),
			val,
			enumConstDesc,
			*n.constVal,
		)
	}

	// in
	if len(n.inVals) > 0 && !slices.Contains(n.inVals, enumVal) {
		return n.violationError(
			"enum.in",
			"value must be in "+formatList(n.inVals),
			val,
			enumInDesc,
			enumVal,
		)
	}

	// not_in
	if len(n.notInVals) > 0 && slices.Contains(n.notInVals, enumVal) {
		return n.violationError(
			"enum.not_in",
			"value must not be in "+formatList(n.notInVals),
			val,
			enumNotInDesc,
			enumVal,
		)
	}

	return nil
}

func (n nativeEnumEval) violationError(
	ruleID string,
	message string,
	fieldValue protoreflect.Value,
	desc protoreflect.FieldDescriptor,
	ruleVal int32,
) error {
	return &ValidationError{Violations: []*Violation{{
		Proto: validate.Violation_builder{
			Field: n.fieldPath(),
			Rule: n.rulePath(validate.FieldPath_builder{
				Elements: []*validate.FieldPathElement{
					fieldPathElement(enumRuleDescriptor),
					fieldPathElement(desc),
				},
			}.Build()),
			RuleId:  proto.String(ruleID),
			Message: proto.String(message),
		}.Build(),
		FieldValue:      fieldValue,
		FieldDescriptor: n.Descriptor,
		RuleValue:       protoreflect.ValueOfInt32(ruleVal),
		RuleDescriptor:  desc,
	}}}
}

func (n nativeEnumEval) Tautology() bool {
	return false
}

var _ evaluator = nativeEnumEval{}

// tryBuildNativeEnumRules attempts to build a native Go evaluator for
// enum const/in/not_in rules. Returns nil if the rules can't be handled
// natively. Note: defined_only is handled separately in enum.go.
func tryBuildNativeEnumRules(base base, rules *validate.EnumRules) evaluator {
	if rules == nil {
		return nil
	}
	if len(rules.ProtoReflect().GetUnknown()) > 0 {
		return nil
	}

	hasRule := false

	var constVal *int32
	if rules.HasConst() {
		constVal = ptr(rules.GetConst())
		hasRule = true
	}

	var inVals []int32
	if inVals = rules.GetIn(); len(inVals) > 0 {
		hasRule = true
	}

	var notInVals []int32
	if notInVals = rules.GetNotIn(); len(notInVals) > 0 {
		hasRule = true
	}

	if !hasRule {
		return nil
	}

	return nativeEnumEval{
		base:      base,
		constVal:  constVal,
		inVals:    inVals,
		notInVals: notInVals,
	}
}
