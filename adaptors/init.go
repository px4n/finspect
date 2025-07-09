package adaptors

import (
	"github.com/px4n/finspect/adaptors/googledrive"
	"github.com/px4n/finspect/adaptors/s3"
)

// RegisterAll registers all available cloud storage providers
func RegisterAll() {
	// Register cloud storage providers
	s3.Register()
	googledrive.Register()
}
