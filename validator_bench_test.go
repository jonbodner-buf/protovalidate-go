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

	pb "buf.build/go/protovalidate/internal/gen/tests/example/v1"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/rodaine/protogofakeit"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func BenchmarkScalar(b *testing.B) {
	benchSuccess(b, &pb.BenchScalar{})
}

func BenchmarkRepeated(b *testing.B) {
	b.Run("Scalar", func(b *testing.B) {
		benchSuccess(b, &pb.BenchRepeatedScalar{})
	})
	b.Run("Message", func(b *testing.B) {
		benchSuccess(b, &pb.BenchRepeatedMessage{})
	})
	b.Run("Unique", func(b *testing.B) {
		b.Run("Scalar", func(b *testing.B) {
			benchSuccess(b, &pb.BenchRepeatedScalarUnique{})
		})
		b.Run("Bytes", func(b *testing.B) {
			benchSuccess(b, &pb.BenchRepeatedBytesUnique{})
		})
	})
}

func BenchmarkMap(b *testing.B) {
	benchSuccess(b, &pb.BenchMap{})
}

func BenchmarkComplexSchema(b *testing.B) {
	benchSuccess(b, &pb.BenchComplexSchema{})
}

func BenchmarkInt32GT(b *testing.B) {
	benchSuccess(b, &pb.BenchGT{})
}

func TestInt32GT(t *testing.T) {
	t.Parallel()
	testSuccess(t, &pb.BenchGT{})
}

func TestComplexSchema(t *testing.T) {
	t.Parallel()
	testSuccess(t, &pb.BenchComplexSchema{})
}

func BenchmarkCompile(b *testing.B) {
	// Measures compile-time allocations for complex schemas
	msg := &pb.BenchComplexSchema{}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = New(WithMessages(msg), WithDisableLazy())
	}
}

func BenchmarkCompileInt32GT(b *testing.B) {
	// Measures compile-time allocations for complex schemas
	msg := &pb.BenchGT{}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = New(WithMessages(msg), WithDisableLazy())
	}
}

func testSuccess(t *testing.T, msg proto.Message) {
	faker := protogofakeit.New(gofakeit.New(1))
	require.NoError(t, faker.FakeProto(msg))
	val, err := New(WithMessages(msg), WithDisableLazy())
	require.NoError(t, err)
	err = val.Validate(msg)
	require.NoError(t, err)
}

func benchSuccess(b *testing.B, msg proto.Message) {
	faker := protogofakeit.New(gofakeit.New(1))
	require.NoError(b, faker.FakeProto(msg))
	val, err := New(WithMessages(msg), WithDisableLazy())
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = val.Validate(msg)
	}
}
