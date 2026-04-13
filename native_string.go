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
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// stringDescriptors bundles the field descriptors for StringRules.
type stringDescriptors struct {
	ruleDesc        protoreflect.FieldDescriptor // FieldRules.string
	constDesc       protoreflect.FieldDescriptor
	lenDesc         protoreflect.FieldDescriptor
	minLenDesc      protoreflect.FieldDescriptor
	maxLenDesc      protoreflect.FieldDescriptor
	lenBytesDesc    protoreflect.FieldDescriptor
	minBytesDesc    protoreflect.FieldDescriptor
	maxBytesDesc    protoreflect.FieldDescriptor
	patternDesc     protoreflect.FieldDescriptor
	prefixDesc      protoreflect.FieldDescriptor
	suffixDesc      protoreflect.FieldDescriptor
	containsDesc    protoreflect.FieldDescriptor
	notContainsDesc protoreflect.FieldDescriptor
	inDesc          protoreflect.FieldDescriptor
	notInDesc       protoreflect.FieldDescriptor
}

//nolint:dupl
func makeStringDescriptors() stringDescriptors {
	rulesDesc := (*validate.StringRules)(nil).ProtoReflect().Descriptor()
	return stringDescriptors{
		ruleDesc:        fieldRulesDesc.Fields().ByName("string"),
		constDesc:       rulesDesc.Fields().ByName("const"),
		lenDesc:         rulesDesc.Fields().ByName("len"),
		minLenDesc:      rulesDesc.Fields().ByName("min_len"),
		maxLenDesc:      rulesDesc.Fields().ByName("max_len"),
		lenBytesDesc:    rulesDesc.Fields().ByName("len_bytes"),
		minBytesDesc:    rulesDesc.Fields().ByName("min_bytes"),
		maxBytesDesc:    rulesDesc.Fields().ByName("max_bytes"),
		patternDesc:     rulesDesc.Fields().ByName("pattern"),
		prefixDesc:      rulesDesc.Fields().ByName("prefix"),
		suffixDesc:      rulesDesc.Fields().ByName("suffix"),
		containsDesc:    rulesDesc.Fields().ByName("contains"),
		notContainsDesc: rulesDesc.Fields().ByName("not_contains"),
		inDesc:          rulesDesc.Fields().ByName("in"),
		notInDesc:       rulesDesc.Fields().ByName("not_in"),
	}
}

//nolint:gochecknoglobals
var strDescs = makeStringDescriptors()

// nativeStringEval is a native Go evaluator for string rules.
// It replaces CEL evaluation with direct Go operations for
// const, in, not_in, len, min_len, max_len, len_bytes, min_bytes,
// max_bytes, pattern, prefix, suffix, contains, and not_contains.
type nativeStringEval struct {
	base
	constVal    *string
	inVals      []string
	notInVals   []string
	exactLen    *uint64
	minLen      *uint64
	maxLen      *uint64
	exactBytes  *uint64
	minBytes    *uint64
	maxBytes    *uint64
	pattern     *regexp.Regexp
	patternStr  string
	prefix      *string
	suffix      *string
	contains    *string
	notContains *string
}

func (n nativeStringEval) Evaluate(_ protoreflect.Message, val protoreflect.Value, _ *validationConfig) error {
	strVal := val.String()

	var runeCount uint64
	if n.exactLen != nil || n.minLen != nil || n.maxLen != nil {
		runeCount = uint64(utf8.RuneCountInString(strVal)) //nolint:gosec
		err := n.evaluateLength(runeCount, val)
		if err != nil {
			return err
		}
	}

	var byteCount uint64
	if n.exactBytes != nil || n.minBytes != nil || n.maxBytes != nil {
		byteCount = uint64(len(strVal))
		err := n.evaluateByteLength(byteCount, val)
		if err != nil {
			return err
		}
	}

	// const
	if n.constVal != nil && strVal != *n.constVal {
		return n.newViolation(strDescs.ruleDesc, strDescs.constDesc,
			"string.const", fmt.Sprintf("value must equal `%s`", *n.constVal),
			val, protoreflect.ValueOfString(*n.constVal))
	}

	// pattern
	if n.pattern != nil && !n.pattern.MatchString(strVal) {
		return n.newViolation(strDescs.ruleDesc, strDescs.patternDesc,
			"string.pattern", fmt.Sprintf("value does not match regex pattern `%s`", n.patternStr),
			val, protoreflect.ValueOfString(n.patternStr))
	}

	// prefix
	if n.prefix != nil && !strings.HasPrefix(strVal, *n.prefix) {
		return n.newViolation(strDescs.ruleDesc, strDescs.prefixDesc,
			"string.prefix", fmt.Sprintf("value does not have prefix `%s`", *n.prefix),
			val, protoreflect.ValueOfString(*n.prefix))
	}

	// suffix
	if n.suffix != nil && !strings.HasSuffix(strVal, *n.suffix) {
		return n.newViolation(strDescs.ruleDesc, strDescs.suffixDesc,
			"string.suffix", fmt.Sprintf("value does not have suffix `%s`", *n.suffix),
			val, protoreflect.ValueOfString(*n.suffix))
	}

	// contains
	if n.contains != nil && !strings.Contains(strVal, *n.contains) {
		return n.newViolation(strDescs.ruleDesc, strDescs.containsDesc,
			"string.contains", fmt.Sprintf("value does not contain substring `%s`", *n.contains),
			val, protoreflect.ValueOfString(*n.contains))
	}

	// not_contains
	if n.notContains != nil && strings.Contains(strVal, *n.notContains) {
		return n.newViolation(strDescs.ruleDesc, strDescs.notContainsDesc,
			"string.not_contains", fmt.Sprintf("value contains substring `%s`", *n.notContains),
			val, protoreflect.ValueOfString(*n.notContains))
	}

	// in
	if len(n.inVals) > 0 && !slices.Contains(n.inVals, strVal) {
		return n.newViolation(strDescs.ruleDesc, strDescs.inDesc,
			"string.in", "value must be in list "+formatStringList(n.inVals),
			val, protoreflect.ValueOfString(strVal))
	}

	// not_in
	if len(n.notInVals) > 0 && slices.Contains(n.notInVals, strVal) {
		return n.newViolation(strDescs.ruleDesc, strDescs.notInDesc,
			"string.not_in", "value must not be in list "+formatStringList(n.notInVals),
			val, protoreflect.ValueOfString(strVal))
	}

	return nil
}

// it would be worse to unify this and evaluateLength than it is to leave them as
// very similar bits of code
//
//nolint:dupl
func (n nativeStringEval) evaluateByteLength(byteCount uint64, val protoreflect.Value) error {
	if n.exactBytes != nil && byteCount != *n.exactBytes {
		return n.newViolation(strDescs.ruleDesc, strDescs.lenBytesDesc,
			"string.len_bytes", fmt.Sprintf("value length must be %d bytes", *n.exactBytes),
			val, protoreflect.ValueOfUint64(*n.exactBytes))
	}
	if n.minBytes != nil && byteCount < *n.minBytes {
		return n.newViolation(strDescs.ruleDesc, strDescs.minBytesDesc,
			"string.min_bytes", fmt.Sprintf("value length must be at least %d bytes", *n.minBytes),
			val, protoreflect.ValueOfUint64(*n.minBytes))
	}
	if n.maxBytes != nil && byteCount > *n.maxBytes {
		return n.newViolation(strDescs.ruleDesc, strDescs.maxBytesDesc,
			"string.max_bytes", fmt.Sprintf("value length must be at most %d bytes", *n.maxBytes),
			val, protoreflect.ValueOfUint64(*n.maxBytes))
	}
	return nil
}

// it would be worse to unify this and evaluateByteLength than it is to leave them as
// very similar bits of code
//
//nolint:dupl
func (n nativeStringEval) evaluateLength(runeCount uint64, val protoreflect.Value) error {
	if n.exactLen != nil && runeCount != *n.exactLen {
		return n.newViolation(strDescs.ruleDesc, strDescs.lenDesc,
			"string.len", fmt.Sprintf("value length must be %d characters", *n.exactLen),
			val, protoreflect.ValueOfUint64(*n.exactLen))
	}
	if n.minLen != nil && runeCount < *n.minLen {
		return n.newViolation(strDescs.ruleDesc, strDescs.minLenDesc,
			"string.min_len", fmt.Sprintf("value length must be at least %d characters", *n.minLen),
			val, protoreflect.ValueOfUint64(*n.minLen))
	}
	if n.maxLen != nil && runeCount > *n.maxLen {
		return n.newViolation(strDescs.ruleDesc, strDescs.maxLenDesc,
			"string.max_len", fmt.Sprintf("value length must be at most %d characters", *n.maxLen),
			val, protoreflect.ValueOfUint64(*n.maxLen))
	}
	return nil
}

func (n nativeStringEval) Tautology() bool {
	return false
}

var _ evaluator = nativeStringEval{}

// tryBuildNativeStringRules attempts to build a native Go evaluator for
// string rules. Returns nil if the rules can't be handled natively.
func tryBuildNativeStringRules(base base, rules *validate.StringRules) evaluator {
	if rules == nil {
		return nil
	}

	// Bail out for custom predefined extensions.
	if len(rules.ProtoReflect().GetUnknown()) > 0 {
		return nil
	}

	// Bail out for well-known format constraints (email, hostname, ip, etc.).
	if rules.HasWellKnown() {
		return nil
	}

	hasRule := false

	var constVal *string
	if rules.HasConst() {
		constVal = ptr(rules.GetConst())
		hasRule = true
	}

	var exactLen *uint64
	if rules.HasLen() {
		exactLen = ptr(rules.GetLen())
		hasRule = true
	}

	var minLen *uint64
	if rules.HasMinLen() {
		minLen = ptr(rules.GetMinLen())
		hasRule = true
	}

	var maxLen *uint64
	if rules.HasMaxLen() {
		maxLen = ptr(rules.GetMaxLen())
		hasRule = true
	}

	var exactBytes *uint64
	if rules.HasLenBytes() {
		exactBytes = ptr(rules.GetLenBytes())
		hasRule = true
	}

	var minBytes *uint64
	if rules.HasMinBytes() {
		minBytes = ptr(rules.GetMinBytes())
		hasRule = true
	}

	var maxBytes *uint64
	if rules.HasMaxBytes() {
		maxBytes = ptr(rules.GetMaxBytes())
		hasRule = true
	}

	var compiledPattern *regexp.Regexp
	var patternStr string
	if rules.HasPattern() {
		patternStr = rules.GetPattern()
		var err error
		compiledPattern, err = regexp.Compile(patternStr)
		if err != nil {
			// Invalid regex — bail to CEL which will also report a CompilationError.
			return nil
		}
		hasRule = true
	}

	var prefix *string
	if rules.HasPrefix() {
		prefix = ptr(rules.GetPrefix())
		hasRule = true
	}

	var suffix *string
	if rules.HasSuffix() {
		suffix = ptr(rules.GetSuffix())
		hasRule = true
	}

	var containsVal *string
	if rules.HasContains() {
		containsVal = ptr(rules.GetContains())
		hasRule = true
	}

	var notContains *string
	if rules.HasNotContains() {
		notContains = ptr(rules.GetNotContains())
		hasRule = true
	}

	var inVals []string
	if inVals = rules.GetIn(); len(inVals) > 0 {
		hasRule = true
	}

	var notInVals []string
	if notInVals = rules.GetNotIn(); len(notInVals) > 0 {
		hasRule = true
	}

	if !hasRule {
		return nil
	}

	return nativeStringEval{
		base:        base,
		constVal:    constVal,
		inVals:      inVals,
		notInVals:   notInVals,
		exactLen:    exactLen,
		minLen:      minLen,
		maxLen:      maxLen,
		exactBytes:  exactBytes,
		minBytes:    minBytes,
		maxBytes:    maxBytes,
		pattern:     compiledPattern,
		patternStr:  patternStr,
		prefix:      prefix,
		suffix:      suffix,
		contains:    containsVal,
		notContains: notContains,
	}
}

// formatStringList formats a []string as [a, b] to match CEL's
// string list formatting (no quoting around elements).
func formatStringList(vals []string) string {
	return "[" + strings.Join(vals, ", ") + "]"
}
