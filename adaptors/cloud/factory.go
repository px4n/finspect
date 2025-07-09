package cloud

import (
	"fmt"
)

// cloudProviderRegistry holds registered cloud adaptors
var cloudProviderRegistry = make(map[Provider]func() Adaptor)

// RegisterProvider registers a cloud adaptor provider
func RegisterProvider(provider Provider, factory func() Adaptor) {
	cloudProviderRegistry[provider] = factory
}

// DefaultCloudAdaptorFactory creates cloud adaptors
type DefaultCloudAdaptorFactory struct{}

// Create creates a cloud adaptor based on provider type
func (f *DefaultCloudAdaptorFactory) Create(provider Provider) (Adaptor, error) {
	factory, ok := cloudProviderRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported cloud provider: %s", provider)
	}

	return factory(), nil
}

// NewFactory creates a new cloud adaptor factory
func NewFactory() AdaptorFactory {
	return &DefaultCloudAdaptorFactory{}
}
