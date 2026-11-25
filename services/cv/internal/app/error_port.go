package app

import (
	"cv/internal/adapters"
	"cv/internal/apperrors"
	"cv/internal/ports"
)

// errorPort serializes every RFC-007 error response. Assign another
// ports.ErrorPort adapter to change the envelope.
var errorPort ports.ErrorPort = adapters.DefaultErrorPort{}

// useErrorPort binds apperrors.Write to errorPort; New calls it when the
// error_port fragment is generated.
func useErrorPort() { apperrors.UseSerializer(errorPort.Serialize) }
