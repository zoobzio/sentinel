package catalog

import (
	"strings"
	"sync"
)

// EventProvider provides event emission capability without catalog importing capitan
type EventProvider interface {
	EmitTypedEvent(typeName string, eventData []byte)
}

var eventProvider EventProvider

// SetEventProvider allows capitan to hydrate catalog with event capability
func SetEventProvider(provider EventProvider) {
	eventProvider = provider
}

// emitEvent emits event through hydrated provider
func emitEvent(typeName string, eventData []byte) {
	if eventProvider != nil {
		eventProvider.EmitTypedEvent(typeName, eventData)
	}
}


// PipzEventHandler handles pipz events for catalog
type PipzEventHandler interface {
	OnProcessorRegistered(contractSignature, keyTypeName, keyValue string)
}

var (
	pipzHandler   PipzEventHandler
	handlerOnce   sync.Once
)

// SetPipzEventHandler allows external systems to register a handler
func SetPipzEventHandler(handler PipzEventHandler) {
	pipzHandler = handler
}

// GetPipzEventHandler returns the current pipz event handler
func GetPipzEventHandler() PipzEventHandler {
	return pipzHandler
}

// AutoRegisterTagsFromBehaviors sets up automatic tag registration
// When behaviors are registered that use specific tags, those tags are auto-registered
func AutoRegisterTagsFromBehaviors() {
	handlerOnce.Do(func() {
		if pipzHandler == nil {
			// Create default handler that auto-registers tags
			pipzHandler = &defaultPipzHandler{}
		}
	})
}

type defaultPipzHandler struct{}

func (h *defaultPipzHandler) OnProcessorRegistered(contractSignature, keyTypeName, keyValue string) {
	// Auto-register tags when security behaviors are registered
	if strings.Contains(contractSignature, "SecurityBehaviorKey") {
		// Check for specific behavior keys that indicate tag usage
		switch keyValue {
		case "field", "field_scope":
			RegisterTag("scope")
		case "encryption", "field_encrypt":
			RegisterTag("encrypt")
		case "redaction", "field_redact":
			RegisterTag("redact")
		}
	}
	
	// Auto-register validate tag for validation behaviors
	if strings.Contains(contractSignature, "ValidationBehaviorKey") {
		RegisterTag("validate")
		// Also check for specific validation types
		switch keyValue {
		case "format", "pattern", "required", "custom":
			RegisterTag("validate")
		}
	}
	
	// Auto-register scope tag for scope behaviors
	if strings.Contains(contractSignature, "ScopeBehaviorKey") {
		RegisterTag("scope")
	}
}