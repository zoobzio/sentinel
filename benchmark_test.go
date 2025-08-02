package sentinel

import (
	"testing"
	"time"
)

// Benchmark struct with various field types and tags.
type BenchmarkStruct struct {
	ID          string                 `json:"id" db:"id" validate:"required,uuid"`
	Name        string                 `json:"name" validate:"required,min=2,max=100"`
	Email       string                 `json:"email" validate:"required,email" encrypt:"pii"`
	Age         int                    `json:"age" validate:"min=0,max=150"`
	Active      bool                   `json:"active" db:"is_active"`
	Score       float64                `json:"score" validate:"min=0,max=100"`
	Tags        []string               `json:"tags" validate:"dive,required"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	Description string                 `json:"description,omitempty" db:"description" validate:"max=1000"`
	Category    string                 `json:"category" validate:"oneof=A B C D E"`
	Priority    int                    `json:"priority" validate:"min=1,max=10"`
	Status      string                 `json:"status" validate:"required,oneof=active inactive pending"`
	Data        []byte                 `json:"data,omitempty" encrypt:"sensitive"`
}

// Simple struct for comparison.
type BenchmarkSimpleStruct struct {
	Value string `json:"value"`
}

func BenchmarkInspectSimple(b *testing.B) {
	s := New().Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkSimpleStruct](s)
	}
}

func BenchmarkInspectComplex(b *testing.B) {
	s := New().Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct](s)
	}
}

func BenchmarkInspectCached(b *testing.B) {
	s := New().Build()

	// Pre-populate cache
	_ = Inspect[BenchmarkStruct](s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct](s)
	}
}

func BenchmarkTagRegistration(b *testing.B) {
	s := New().Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Tag("custom")
	}
}

func BenchmarkPolicyApplication(b *testing.B) {
	policy := Policy{
		Name: "test-policy",
		Policies: []TypePolicy{
			{
				Match: "*Struct",
				Fields: []FieldPolicy{
					{
						Match: "Email",
						Apply: map[string]string{
							"encrypt": "pii",
						},
					},
				},
			},
		},
	}

	s := New().WithPolicy(policy).Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct](s)
	}
}

func BenchmarkConcurrentInspect(b *testing.B) {
	s := New().Build()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Inspect[BenchmarkStruct](s)
		}
	})
}

func BenchmarkInspectMemory(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := New().Build()
		_ = Inspect[BenchmarkStruct](s)
	}
}
