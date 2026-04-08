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
	int32RuleDescriptor = fieldRulesDesc.Fields().ByName("int32")
	int32RulesDesc      = (&validate.Int32Rules{}).ProtoReflect().Descriptor()
	int32GtDescriptor   = int32RulesDesc.Fields().ByName("gt")
	int32GteDescriptor  = int32RulesDesc.Fields().ByName("gte")
	int32LtDescriptor   = int32RulesDesc.Fields().ByName("lt")
	int32LteDescriptor  = int32RulesDesc.Fields().ByName("lte")
)

// lowerBound describes which lower bound constraint is active.
type lowerBound int

const (
	lowerBoundNone lowerBound = iota
	// lowerBoundGte is an inclusive lower bound (>=).
	lowerBoundGte
	// lowerBoundGt is an exclusive lower bound (>).
	lowerBoundGt
)

// upperBound describes which upper bound constraint is active.
type upperBound int

const (
	upperBoundNone upperBound = iota
	upperBoundLt
	upperBoundLte
)

// nativeInt32Compare is a native Go evaluator for int32 gt/gte rules
// (with optional lt/lte combinations). It replaces CEL evaluation
// with direct Go comparisons.
type nativeInt32Compare struct {
	base
	lo    int32      // lower bound value (gt or gte threshold)
	lower lowerBound // gt (exclusive) or gte (inclusive)
	hi    int32      // upper bound value (lt or lte threshold)
	upper upperBound // none, lt, or lte
}

// belowLo reports whether v violates the lower bound.
func (n nativeInt32Compare) belowLo(v int32) bool {
	if n.lower == lowerBoundGt {
		return v <= n.lo
	}
	return v < n.lo
}

// aboveHi reports whether v violates the upper bound.
func (n nativeInt32Compare) aboveHi(v int32) bool {
	if n.upper == upperBoundLt {
		return v >= n.hi
	}
	return v > n.hi
}

// isNormalRange reports whether lo and hi form a normal (non-exclusive) range.
// In CEL, all four combinations (gt/gte × lt/lte) use >= for this check.
func (n nativeInt32Compare) isNormalRange() bool {
	return n.hi >= n.lo
}

func (n nativeInt32Compare) loDesc() protoreflect.FieldDescriptor {
	if n.lower == lowerBoundGt {
		return int32GtDescriptor
	}
	return int32GteDescriptor
}

func (n nativeInt32Compare) hiDesc() protoreflect.FieldDescriptor {
	if n.upper == upperBoundLt {
		return int32LtDescriptor
	}
	return int32LteDescriptor
}

func (n nativeInt32Compare) gtRulePrefix() string {
	if n.lower == lowerBoundGt {
		return "int32.gt"
	}
	return "int32.gte"
}

func (n nativeInt32Compare) ltRulePrefix() string {
	if n.upper == upperBoundLt {
		return "int32.lt"
	}
	return "int32.lte"
}

func (n nativeInt32Compare) rule() string {
	if n.lower != lowerBoundNone {
		prefix := n.gtRulePrefix()
		switch n.upper {
		case upperBoundLt:
			prefix += "_lt"
			if !n.isNormalRange() {
				prefix += "_exclusive"
			}
		case upperBoundLte:
			prefix += "_lte"
			if !n.isNormalRange() {
				prefix += "_exclusive"
			}
		}
		return prefix
	}
	return n.ltRulePrefix()
}

func (n nativeInt32Compare) loMessage() string {
	if n.lower == lowerBoundGt {
		return fmt.Sprintf("greater than %d", n.lo)
	}
	return fmt.Sprintf("greater than or equal to %d", n.lo)
}

func (n nativeInt32Compare) hiMessage() string {
	if n.upper == upperBoundLt {
		return fmt.Sprintf("less than %d", n.hi)
	}
	return fmt.Sprintf("less than or equal to %d", n.hi)
}

func (n nativeInt32Compare) conjunction() string {
	if n.isNormalRange() {
		return "and"
	}
	return "or"
}

func (n nativeInt32Compare) Evaluate(_ protoreflect.Message, val protoreflect.Value, _ *validationConfig) error {
	// this will always be an int32, no concern about overflow.
	int32Val := int32(val.Int()) //nolint:gosec

	/*
		considerations:
		- normal range or not
		- gt/gte set
		- lt/lte set
	*/
	switch {
	case n.lower == lowerBoundNone:
		if n.aboveHi(int32Val) {
			return n.violationErr(
				n.rule(),
				"value must be "+n.hiMessage(),
				val,
				n.hiDesc(),
				n.hi,
			)
		}
	case n.upper == upperBoundNone:
		if n.belowLo(int32Val) {
			return n.violationErr(
				n.rule(),
				"value must be "+n.loMessage(),
				val,
				n.loDesc(),
				n.lo,
			)
		}
	default:
		var failure bool
		if n.isNormalRange() {
			failure = n.aboveHi(int32Val) || n.belowLo(int32Val)
		} else {
			failure = n.aboveHi(int32Val) && n.belowLo(int32Val)
		}
		if failure {
			return n.violationErr(
				n.rule(),
				fmt.Sprintf("value must be %s %s %s", n.loMessage(), n.conjunction(), n.hiMessage()),
				val,
				n.loDesc(),
				n.lo,
			)
		}
	}
	return nil
}

func (n nativeInt32Compare) violationErr(
	ruleID string,
	message string,
	fieldValue protoreflect.Value,
	desc protoreflect.FieldDescriptor,
	valInt32 int32,
) error {
	return &ValidationError{Violations: []*Violation{{
		Proto: validate.Violation_builder{
			Field: n.fieldPath(),
			Rule: n.rulePath(validate.FieldPath_builder{
				Elements: []*validate.FieldPathElement{
					fieldPathElement(int32RuleDescriptor),
					fieldPathElement(desc),
				},
			}.Build()),
			RuleId:  proto.String(ruleID),
			Message: proto.String(message),
		}.Build(),
		FieldValue:      fieldValue,
		FieldDescriptor: n.Descriptor,
		RuleValue:       protoreflect.ValueOfInt32(valInt32),
		RuleDescriptor:  desc,
	}}}
}

func (n nativeInt32Compare) Tautology() bool {
	return false
}

var _ evaluator = nativeInt32Compare{}

// tryBuildNativeInt32Rules attempts to build a native Go evaluator for
// int32 rules. Returns nil if the rules can't be handled natively.
// This includes cases with const, in, not_in, or any extension fields
// (custom predefined rules) — those must go through CEL.
func tryBuildNativeInt32Rules(
	base base,
	rules *validate.Int32Rules,
) evaluator {
	if rules == nil {
		return nil
	}
	if rules.HasConst() || len(rules.GetIn()) > 0 || len(rules.GetNotIn()) > 0 {
		return nil
	}
	// Bail out if the rules message has unknown fields, which indicate
	// custom predefined extensions that we can't handle natively.
	if len(rules.ProtoReflect().GetUnknown()) > 0 {
		return nil
	}

	var lowerValue int32
	lower := lowerBoundNone
	switch {
	case rules.HasGt():
		lower = lowerBoundGt
		lowerValue = rules.GetGt()
	case rules.HasGte():
		lower = lowerBoundGte
		lowerValue = rules.GetGte()
	}

	var upperValue int32
	upper := upperBoundNone
	switch {
	case rules.HasLt():
		upper = upperBoundLt
		upperValue = rules.GetLt()
	case rules.HasLte():
		upper = upperBoundLte
		upperValue = rules.GetLte()
	}

	// if we got here and don't have any bounds, we can't handle this.
	if lower == lowerBoundNone && upper == upperBoundNone {
		return nil
	}

	return nativeInt32Compare{
		base:  base,
		lo:    lowerValue,
		lower: lower,
		hi:    upperValue,
		upper: upper,
	}
}
