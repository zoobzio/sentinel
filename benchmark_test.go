package catalog

import (
	"reflect"
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

// Large struct with many fields.
type LargeStruct struct {
	Field001 string `json:"field_001" validate:"required"`
	Field002 string `json:"field_002" validate:"required"`
	Field003 string `json:"field_003" validate:"required"`
	Field004 string `json:"field_004" validate:"required"`
	Field005 string `json:"field_005" validate:"required"`
	Field006 string `json:"field_006" validate:"required"`
	Field007 string `json:"field_007" validate:"required"`
	Field008 string `json:"field_008" validate:"required"`
	Field009 string `json:"field_009" validate:"required"`
	Field010 string `json:"field_010" validate:"required"`
	Field011 string `json:"field_011" validate:"required"`
	Field012 string `json:"field_012" validate:"required"`
	Field013 string `json:"field_013" validate:"required"`
	Field014 string `json:"field_014" validate:"required"`
	Field015 string `json:"field_015" validate:"required"`
	Field016 string `json:"field_016" validate:"required"`
	Field017 string `json:"field_017" validate:"required"`
	Field018 string `json:"field_018" validate:"required"`
	Field019 string `json:"field_019" validate:"required"`
	Field020 string `json:"field_020" validate:"required"`
	Field021 string `json:"field_021" validate:"required"`
	Field022 string `json:"field_022" validate:"required"`
	Field023 string `json:"field_023" validate:"required"`
	Field024 string `json:"field_024" validate:"required"`
	Field025 string `json:"field_025" validate:"required"`
	Field026 string `json:"field_026" validate:"required"`
	Field027 string `json:"field_027" validate:"required"`
	Field028 string `json:"field_028" validate:"required"`
	Field029 string `json:"field_029" validate:"required"`
	Field030 string `json:"field_030" validate:"required"`
	Field031 string `json:"field_031" validate:"required"`
	Field032 string `json:"field_032" validate:"required"`
	Field033 string `json:"field_033" validate:"required"`
	Field034 string `json:"field_034" validate:"required"`
	Field035 string `json:"field_035" validate:"required"`
	Field036 string `json:"field_036" validate:"required"`
	Field037 string `json:"field_037" validate:"required"`
	Field038 string `json:"field_038" validate:"required"`
	Field039 string `json:"field_039" validate:"required"`
	Field040 string `json:"field_040" validate:"required"`
	Field041 string `json:"field_041" validate:"required"`
	Field042 string `json:"field_042" validate:"required"`
	Field043 string `json:"field_043" validate:"required"`
	Field044 string `json:"field_044" validate:"required"`
	Field045 string `json:"field_045" validate:"required"`
	Field046 string `json:"field_046" validate:"required"`
	Field047 string `json:"field_047" validate:"required"`
	Field048 string `json:"field_048" validate:"required"`
	Field049 string `json:"field_049" validate:"required"`
	Field050 string `json:"field_050" validate:"required"`
}

func BenchmarkInspectSimple(b *testing.B) {
	// Clear cache before benchmark
	cacheMutex.Lock()
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkSimpleStruct]()
	}
}

func BenchmarkInspectComplex(b *testing.B) {
	// Clear cache before benchmark
	cacheMutex.Lock()
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()
	}
}

func BenchmarkInspectLarge(b *testing.B) {
	// Clear cache before benchmark
	cacheMutex.Lock()
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[LargeStruct]()
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

func BenchmarkInspectPointer(b *testing.B) {
	// Clear cache before benchmark
	cacheMutex.Lock()
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[*BenchmarkStruct]()
	}
}

func BenchmarkTagRegistration(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Tag("custom")
	}
}

func BenchmarkBrowse(b *testing.B) {
	// Pre-populate cache with various types
	_ = Inspect[BenchmarkSimpleStruct]()
	_ = Inspect[BenchmarkStruct]()
	_ = Inspect[LargeStruct]()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Browse()
	}
}

func BenchmarkExtractMetadata(b *testing.B) {
	typ := reflect.TypeOf(BenchmarkStruct{})
	var zero BenchmarkStruct

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = extractMetadata(typ, zero)
	}
}

func BenchmarkExtractFieldMetadata(b *testing.B) {
	typ := reflect.TypeOf(BenchmarkStruct{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = extractFieldMetadata(typ)
	}
}

func BenchmarkGetModelMetadata(b *testing.B) {
	// Pre-populate cache
	metadata := ModelMetadata{
		TypeName:    "BenchmarkType",
		PackageName: "catalog",
		Fields:      []FieldMetadata{{Name: "Field1"}},
	}

	cacheMutex.Lock()
	metadataCache["BenchmarkType"] = metadata
	cacheMutex.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = GetModelMetadata("BenchmarkType")
	}
}

func BenchmarkConcurrentInspect(b *testing.B) {
	// Clear cache before benchmark
	cacheMutex.Lock()
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex.Unlock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Inspect[BenchmarkStruct]()
		}
	})
}

func BenchmarkConcurrentBrowse(b *testing.B) {
	// Pre-populate cache
	_ = Inspect[BenchmarkSimpleStruct]()
	_ = Inspect[BenchmarkStruct]()
	_ = Inspect[LargeStruct]()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Browse()
		}
	})
}

func BenchmarkConcurrentTag(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Tag("tag")
			i++
		}
	})
}

// Memory allocation benchmarks.
func BenchmarkInspectMemory(b *testing.B) {
	// Clear cache to measure fresh extraction
	cacheMutex.Lock()
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex.Unlock()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()

		// Clear cache each iteration to measure extraction memory
		cacheMutex.Lock()
		metadataCache = make(map[string]ModelMetadata)
		cacheMutex.Unlock()
	}
}

func BenchmarkInspectCachedMemory(b *testing.B) {
	// Pre-populate cache
	_ = Inspect[BenchmarkStruct]()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Inspect[BenchmarkStruct]()
	}
}
