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

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//nolint:gochecknoglobals
var (
	boolRuleDesc  = fieldRulesDesc.Fields().ByName("bool")
	boolConstDesc = (*validate.BoolRules)(nil).ProtoReflect().Descriptor().Fields().ByName("const")
)

// nativeBoolEval is a native Go evaluator for bool const rules.
type nativeBoolEval struct {
	base
	constVal bool
}

func (n nativeBoolEval) Evaluate(_ protoreflect.Message, val protoreflect.Value, _ *validationConfig) error {
	if val.Bool() != n.constVal {
		return &ValidationError{Violations: []*Violation{{
			Proto: validate.Violation_builder{
				Field: n.fieldPath(),
				Rule: n.rulePath(validate.FieldPath_builder{
					Elements: []*validate.FieldPathElement{
						fieldPathElement(boolRuleDesc),
						fieldPathElement(boolConstDesc),
					},
				}.Build()),
				RuleId:  proto.String("bool.const"),
				Message: proto.String(fmt.Sprintf("value must equal %t", n.constVal)),
			}.Build(),
			FieldValue:      val,
			FieldDescriptor: n.Descriptor,
			RuleValue:       protoreflect.ValueOfBool(n.constVal),
			RuleDescriptor:  boolConstDesc,
		}}}
	}
	return nil
}

func (n nativeBoolEval) Tautology() bool {
	return false
}

var _ evaluator = nativeBoolEval{}

// tryBuildNativeBoolRules attempts to build a native Go evaluator for
// bool rules. Returns nil if the rules can't be handled natively.
func tryBuildNativeBoolRules(base base, rules *validate.BoolRules) evaluator {
	if rules == nil {
		return nil
	}
	if len(rules.ProtoReflect().GetUnknown()) > 0 {
		return nil
	}
	if !rules.HasConst() {
		return nil
	}
	return nativeBoolEval{
		base:     base,
		constVal: rules.GetConst(),
	}
}
