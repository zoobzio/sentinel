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

// Setup admin for benchmarks.
func init() {
	// Set up sealed configuration for benchmarks
	resetAdminForTesting()
	admin, err := NewAdmin()
	if err != nil {
		panic("failed to create admin for benchmarks: " + err.Error())
	}
	if err := admin.Seal(); err != nil {
		panic(err)
	}
}

func BenchmarkInspectSimple(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkSimpleStruct]()
	}
}

func BenchmarkInspectComplex(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()
	}
}

func BenchmarkInspectCached(b *testing.B) {
	// Pre-populate cache
	_ = Inspect[BenchmarkStruct]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()
	}
}

func BenchmarkTagRegistration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Tag("custom")
	}
}

func BenchmarkPolicyApplication(b *testing.B) {
	// Note: With global singleton, policies would need to be applied differently
	// This benchmark is now just measuring base performance
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()
	}
}

func BenchmarkConcurrentInspect(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Inspect[BenchmarkStruct]()
		}
	})
}

func BenchmarkInspectMemory(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()
	}
}
