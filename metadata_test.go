package sentinel

import (
	"reflect"
	"testing"
)

func TestCodec(t *testing.T) {
	t.Run("codec constants", func(t *testing.T) {
		// Verify codec constants have expected values
		codecs := map[Codec]string{
			CodecJSON:       "json",
			CodecXML:        "xml",
			CodecYAML:       "yaml",
			CodecTOML:       "toml",
			CodecMsgpack:    "msgpack",
			CodecProtobuf:   "protobuf",
			CodecCBOR:       "cbor",
			CodecGOB:        "gob",
			CodecCSV:        "csv",
			CodecAvro:       "avro",
			CodecThrift:     "thrift",
			CodecBSON:       "bson",
			CodecFlatbuffer: "flatbuffer",
			CodecCapnProto:  "capnproto",
			CodecIon:        "ion",
		}

		for codec, expected := range codecs {
			if string(codec) != expected {
				t.Errorf("codec %v: expected %q, got %q", codec, expected, string(codec))
			}
		}
	})

	t.Run("IsValidCodec", func(t *testing.T) {
		// Test valid codecs
		validTests := []string{
			"json", "xml", "yaml", "toml", "msgpack", "protobuf",
			"cbor", "gob", "csv", "avro", "thrift", "bson",
			"flatbuffer", "capnproto", "ion",
		}

		for _, codec := range validTests {
			if !IsValidCodec(codec) {
				t.Errorf("expected %q to be valid codec", codec)
			}
		}

		// Test invalid codecs
		invalidTests := []string{
			"", "JSON", "Json", "unknown", "binary", "text",
			"jsonp", "xml2", "yaml2", "proto", " json", "json ",
		}

		for _, codec := range invalidTests {
			if IsValidCodec(codec) {
				t.Errorf("expected %q to be invalid codec", codec)
			}
		}
	})
}

func TestModelMetadata(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		metadata := ModelMetadata{
			TypeName:    "User",
			PackageName: "main",
			Fields: []FieldMetadata{
				{
					Name: "ID",
					Type: "string",
					Tags: map[string]string{"json": "id"},
				},
			},
			Codecs: []string{"json", "xml"},
		}

		if metadata.TypeName != "User" {
			t.Errorf("expected TypeName 'User', got %s", metadata.TypeName)
		}
		if metadata.PackageName != "main" {
			t.Errorf("expected PackageName 'main', got %s", metadata.PackageName)
		}
		if len(metadata.Fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(metadata.Fields))
		}
		if len(metadata.Codecs) != 2 {
			t.Errorf("expected 2 codecs, got %d", len(metadata.Codecs))
		}
	})

	t.Run("json tags", func(t *testing.T) {
		// Verify JSON struct tags are properly defined
		metadata := ModelMetadata{}
		metaType := reflect.TypeOf(metadata)

		expectedTags := map[string]string{
			"TypeName":    "type_name",
			"PackageName": "package_name",
			"Fields":      "fields",
			"Codecs":      "codecs,omitempty",
		}

		for fieldName, expectedTag := range expectedTags {
			field, found := metaType.FieldByName(fieldName)
			if !found {
				t.Errorf("field %s not found", fieldName)
				continue
			}
			if tag := field.Tag.Get("json"); tag != expectedTag {
				t.Errorf("field %s: expected json tag %q, got %q", fieldName, expectedTag, tag)
			}
		}
	})
}

func TestFieldMetadata(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		field := FieldMetadata{
			Name: "Email",
			Type: "string",
			Tags: map[string]string{
				"json":     "email",
				"validate": "required,email",
				"encrypt":  "pii",
			},
		}

		if field.Name != "Email" {
			t.Errorf("expected Name 'Email', got %s", field.Name)
		}
		if field.Type != "string" {
			t.Errorf("expected Type 'string', got %s", field.Type)
		}
		if len(field.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(field.Tags))
		}
		if field.Tags["json"] != "email" {
			t.Errorf("expected json tag 'email', got %s", field.Tags["json"])
		}
	})

	t.Run("json tags", func(t *testing.T) {
		// Verify JSON struct tags are properly defined
		field := FieldMetadata{}
		fieldType := reflect.TypeOf(field)

		expectedTags := map[string]string{
			"Tags": "tags,omitempty",
			"Name": "name",
			"Type": "type",
		}

		for fieldName, expectedTag := range expectedTags {
			f, found := fieldType.FieldByName(fieldName)
			if !found {
				t.Errorf("field %s not found", fieldName)
				continue
			}
			if tag := f.Tag.Get("json"); tag != expectedTag {
				t.Errorf("field %s: expected json tag %q, got %q", fieldName, expectedTag, tag)
			}
		}
	})

	t.Run("nil tags map", func(_ *testing.T) {
		_ = FieldMetadata{
			Name: "ID",
			Type: "int",
			Tags: nil,
		}

		// Should not panic.
		// When Tags is nil, this is expected and allowed behavior.
	})
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		{
			name:     "nil type",
			input:    nil,
			expected: "nil",
		},
		{
			name:     "string type",
			input:    reflect.TypeOf(""),
			expected: "string",
		},
		{
			name:     "int type",
			input:    reflect.TypeOf(0),
			expected: "int",
		},
		{
			name:     "struct type",
			input:    reflect.TypeOf(struct{ Name string }{}),
			expected: "",
		},
		{
			name:     "named struct type",
			input:    reflect.TypeOf(ModelMetadata{}),
			expected: "ModelMetadata",
		},
		{
			name:     "pointer to struct",
			input:    reflect.TypeOf(&ModelMetadata{}),
			expected: "ModelMetadata",
		},
		{
			name:     "pointer to string",
			input:    reflect.TypeOf((*string)(nil)),
			expected: "string",
		},
		{
			name:     "slice type",
			input:    reflect.TypeOf([]string{}),
			expected: "",
		},
		{
			name:     "map type",
			input:    reflect.TypeOf(map[string]int{}),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidCodecsMap(t *testing.T) {
	// Ensure validCodecs map has all codec constants
	expectedCount := 15 // Number of codec constants

	if len(validCodecs) != expectedCount {
		t.Errorf("expected %d codecs in validCodecs map, got %d", expectedCount, len(validCodecs))
	}

	// Verify each codec constant is in the map
	codecs := []Codec{
		CodecJSON, CodecXML, CodecYAML, CodecTOML, CodecMsgpack,
		CodecProtobuf, CodecCBOR, CodecGOB, CodecCSV, CodecAvro,
		CodecThrift, CodecBSON, CodecFlatbuffer, CodecCapnProto, CodecIon,
	}

	for _, codec := range codecs {
		if !validCodecs[string(codec)] {
			t.Errorf("codec %s not found in validCodecs map", codec)
		}
	}
}
