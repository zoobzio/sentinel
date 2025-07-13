package catalog

import (
	"fmt"
	"reflect"
	"sync"
)

// SetupConvention is the standard interface for type initialization.
// Types can implement this to perform any setup when first seen by catalog.
type SetupConvention interface {
	Setup()
}

// TypeConventionCheck defines what to check when a type is first ingested
type TypeConventionCheck struct {
	// Name of the convention (for error messages)
	Name string
	// Function to check if this convention is required for the given metadata
	IsRequired func(metadata ModelMetadata) bool
	// Interface pointer (e.g., (*security.SecurityConvention)(nil))
	InterfacePtr interface{}
	// Error message if required but not implemented
	FailureMessage string
}

// TypeConventionChecker is a function that returns a convention check for a given type
type TypeConventionChecker func(metadata ModelMetadata) *TypeConventionCheck

var (
	typeConventionsMutex    sync.RWMutex
	typeConventionCheckers  []TypeConventionChecker
	checkedTypes           = make(map[string]bool) // Track which types we've already checked
)

// RegisterTypeConvention registers a convention checker that will be called
// when types are first ingested via Select[T]
func RegisterTypeConvention(checker TypeConventionChecker) {
	typeConventionsMutex.Lock()
	defer typeConventionsMutex.Unlock()
	
	typeConventionCheckers = append(typeConventionCheckers, checker)
}

// checkTypeConventions checks all registered conventions for a type
// This is called from Select[T] when a type is first seen
func checkTypeConventions[T any](metadata ModelMetadata) {
	typeName := metadata.TypeName
	
	// Check if we've already processed this type
	typeConventionsMutex.RLock()
	if checkedTypes[typeName] {
		typeConventionsMutex.RUnlock()
		return
	}
	typeConventionsMutex.RUnlock()
	
	// Mark as checked
	typeConventionsMutex.Lock()
	checkedTypes[typeName] = true
	checkers := make([]TypeConventionChecker, len(typeConventionCheckers))
	copy(checkers, typeConventionCheckers)
	typeConventionsMutex.Unlock()
	
	var zero T
	zeroValue := reflect.ValueOf(&zero).Elem().Interface()
	
	// First, always check for Setup() convention
	if setup, ok := zeroValue.(SetupConvention); ok {
		setup.Setup()
	}
	
	// Then check adapter-specific conventions
	
	for _, checker := range checkers {
		check := checker(metadata)
		if check == nil {
			continue
		}
		
		if check.IsRequired(metadata) {
			// Check if type implements the interface
			interfaceType := reflect.TypeOf(check.InterfacePtr).Elem()
			if !reflect.TypeOf(zeroValue).Implements(interfaceType) {
				panic(fmt.Sprintf("catalog: type %s %s", typeName, check.FailureMessage))
			}
			
			// Type implements the interface, call the convention method
			// For methods with no parameters and no return values
			method := reflect.ValueOf(zeroValue).MethodByName(getMethodName(interfaceType))
			if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 0 {
				method.Call(nil)
			}
		}
	}
}

// getMethodName extracts the method name from an interface type
// For now, we assume single-method interfaces
func getMethodName(interfaceType reflect.Type) string {
	if interfaceType.NumMethod() > 0 {
		return interfaceType.Method(0).Name
	}
	return ""
}