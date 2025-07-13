package catalog

import (
	"fmt"
	"reflect"
	"time"
)

// FieldType represents the type of a struct field
type FieldType int

const (
	UnknownType FieldType = iota
	StringType
	IntType
	Int64Type
	Float64Type
	BoolType
	TimeType
	BytesType
	
	// Slice types
	StringSliceType
	IntSliceType
	Int64SliceType
	Float64SliceType
	BoolSliceType
	
	// Pointer types (nullable)
	StringPtrType
	IntPtrType
	Int64PtrType
	Float64PtrType
	BoolPtrType
	TimePtrType
)

// FieldManipulator provides type-safe field access without reflection in hot paths
type FieldManipulator[T any] struct {
	fieldName   string
	fieldType   FieldType
	getValue    func(T) reflect.Value
	setValue    func(*T, reflect.Value)
}

// Type returns the field type
func (m *FieldManipulator[T]) Type() FieldType {
	return m.fieldType
}

// Name returns the field name
func (m *FieldManipulator[T]) Name() string {
	return m.fieldName
}

// GetString gets a string field value
func (m *FieldManipulator[T]) GetString(src T) (string, error) {
	if m.fieldType != StringType {
		return "", fmt.Errorf("field %s is %v, not string", m.fieldName, m.fieldType)
	}
	return m.getValue(src).String(), nil
}

// SetString sets a string field value
func (m *FieldManipulator[T]) SetString(dest *T, val string) error {
	if m.fieldType != StringType {
		return fmt.Errorf("field %s is %v, not string", m.fieldName, m.fieldType)
	}
	m.setValue(dest, reflect.ValueOf(val))
	return nil
}

// GetInt gets an int field value
func (m *FieldManipulator[T]) GetInt(src T) (int, error) {
	if m.fieldType != IntType {
		return 0, fmt.Errorf("field %s is %v, not int", m.fieldName, m.fieldType)
	}
	return int(m.getValue(src).Int()), nil
}

// SetInt sets an int field value
func (m *FieldManipulator[T]) SetInt(dest *T, val int) error {
	if m.fieldType != IntType {
		return fmt.Errorf("field %s is %v, not int", m.fieldName, m.fieldType)
	}
	m.setValue(dest, reflect.ValueOf(val))
	return nil
}

// GetBool gets a bool field value
func (m *FieldManipulator[T]) GetBool(src T) (bool, error) {
	if m.fieldType != BoolType {
		return false, fmt.Errorf("field %s is %v, not bool", m.fieldName, m.fieldType)
	}
	return m.getValue(src).Bool(), nil
}

// SetBool sets a bool field value
func (m *FieldManipulator[T]) SetBool(dest *T, val bool) error {
	if m.fieldType != BoolType {
		return fmt.Errorf("field %s is %v, not bool", m.fieldName, m.fieldType)
	}
	m.setValue(dest, reflect.ValueOf(val))
	return nil
}

// Redact sets the field to its redacted value based on type
func (m *FieldManipulator[T]) Redact(dest *T) error {
	switch m.fieldType {
	case StringType:
		return m.SetString(dest, "[REDACTED]")
	case IntType:
		return m.SetInt(dest, -1)
	case Int64Type:
		m.setValue(dest, reflect.ValueOf(int64(-1)))
		return nil
	case Float64Type:
		m.setValue(dest, reflect.ValueOf(float64(-1.0)))
		return nil
	case BoolType:
		return m.SetBool(dest, false)
	case TimeType:
		m.setValue(dest, reflect.ValueOf(time.Time{}))
		return nil
	case BytesType:
		m.setValue(dest, reflect.ValueOf([]byte(nil)))
		return nil
		
	// Slices become empty
	case StringSliceType:
		m.setValue(dest, reflect.ValueOf([]string{}))
		return nil
	case IntSliceType:
		m.setValue(dest, reflect.ValueOf([]int{}))
		return nil
	case Int64SliceType:
		m.setValue(dest, reflect.ValueOf([]int64{}))
		return nil
	case Float64SliceType:
		m.setValue(dest, reflect.ValueOf([]float64{}))
		return nil
	case BoolSliceType:
		m.setValue(dest, reflect.ValueOf([]bool{}))
		return nil
		
	// Pointers become nil
	case StringPtrType, IntPtrType, Int64PtrType, Float64PtrType, BoolPtrType, TimePtrType:
		m.setValue(dest, reflect.Zero(reflect.TypeOf(m.getValue(*new(T)).Interface())))
		return nil
		
	default:
		return fmt.Errorf("unsupported field type %v for redaction", m.fieldType)
	}
}

// SetNull sets the field to its zero value
func (m *FieldManipulator[T]) SetNull(dest *T) error {
	zeroValue := reflect.Zero(reflect.TypeOf(m.getValue(*new(T)).Interface()))
	m.setValue(dest, zeroValue)
	return nil
}

// Helper function to determine FieldType from reflect.Type
func getFieldType(t reflect.Type) FieldType {
	switch t.Kind() {
	case reflect.String:
		return StringType
	case reflect.Int:
		return IntType
	case reflect.Int64:
		return Int64Type
	case reflect.Float64:
		return Float64Type
	case reflect.Bool:
		return BoolType
	case reflect.Slice:
		switch t.Elem().Kind() {
		case reflect.String:
			return StringSliceType
		case reflect.Int:
			return IntSliceType
		case reflect.Int64:
			return Int64SliceType
		case reflect.Float64:
			return Float64SliceType
		case reflect.Bool:
			return BoolSliceType
		case reflect.Uint8: // []byte
			return BytesType
		}
	case reflect.Ptr:
		switch t.Elem().Kind() {
		case reflect.String:
			return StringPtrType
		case reflect.Int:
			return IntPtrType
		case reflect.Int64:
			return Int64PtrType
		case reflect.Float64:
			return Float64PtrType
		case reflect.Bool:
			return BoolPtrType
		}
	case reflect.Struct:
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return TimeType
		}
	}
	
	return UnknownType
}

// buildFieldManipulators creates manipulators for all fields of type T
func buildFieldManipulators[T any]() map[string]*FieldManipulator[T] {
	manipulators := make(map[string]*FieldManipulator[T])
	
	// Get the type
	var zero T
	t := reflect.TypeOf(zero)
	
	// Handle if T is already a pointer type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		// Capture loop variables
		fieldIndex := i
		fieldType := getFieldType(field.Type)
		
		// Skip unsupported types
		if fieldType == UnknownType {
			continue
		}
		
		manipulator := &FieldManipulator[T]{
			fieldName: field.Name,
			fieldType: fieldType,
			
			getValue: func(src T) reflect.Value {
				v := reflect.ValueOf(src)
				// If T is a pointer type, we need Elem()
				if v.Kind() == reflect.Ptr {
					if v.IsNil() {
						// Return zero value for the field type
						return reflect.Zero(field.Type)
					}
					v = v.Elem()
				}
				return v.Field(fieldIndex)
			},
			
			setValue: func(dest *T, val reflect.Value) {
				v := reflect.ValueOf(dest).Elem() // dest is always *T
				v.Field(fieldIndex).Set(val)
			},
		}
		
		manipulators[field.Name] = manipulator
	}
	
	return manipulators
}