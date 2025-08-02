package sentinel

import (
	"reflect"
)

// Codec represents a supported serialization format.
type Codec string

// Supported serialization codecs.
const (
	CodecJSON       Codec = "json"
	CodecXML        Codec = "xml"
	CodecYAML       Codec = "yaml"
	CodecTOML       Codec = "toml"
	CodecMsgpack    Codec = "msgpack"
	CodecProtobuf   Codec = "protobuf"
	CodecCBOR       Codec = "cbor"
	CodecGOB        Codec = "gob"
	CodecCSV        Codec = "csv"
	CodecAvro       Codec = "avro"
	CodecThrift     Codec = "thrift"
	CodecBSON       Codec = "bson"
	CodecFlatbuffer Codec = "flatbuffer"
	CodecCapnProto  Codec = "capnproto"
	CodecIon        Codec = "ion"
)

// validCodecs contains all valid codec values for validation.
var validCodecs = map[string]bool{
	string(CodecJSON):       true,
	string(CodecXML):        true,
	string(CodecYAML):       true,
	string(CodecTOML):       true,
	string(CodecMsgpack):    true,
	string(CodecProtobuf):   true,
	string(CodecCBOR):       true,
	string(CodecGOB):        true,
	string(CodecCSV):        true,
	string(CodecAvro):       true,
	string(CodecThrift):     true,
	string(CodecBSON):       true,
	string(CodecFlatbuffer): true,
	string(CodecCapnProto):  true,
	string(CodecIon):        true,
}

// IsValidCodec checks if a codec string is valid.
func IsValidCodec(codec string) bool {
	return validCodecs[codec]
}

// ModelMetadata contains comprehensive information about a user model.
type ModelMetadata struct {
	TypeName    string          `json:"type_name"`
	PackageName string          `json:"package_name"`
	Fields      []FieldMetadata `json:"fields"`
	Codecs      []string        `json:"codecs,omitempty"`
}

// FieldMetadata captures field-level information and all struct tags.
type FieldMetadata struct {
	Tags map[string]string `json:"tags,omitempty"`
	Name string            `json:"name"`
	Type string            `json:"type"`
}

// getTypeName extracts the type name from a reflect.Type.
func getTypeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return "nil"
	}
	return t.Name()
}
