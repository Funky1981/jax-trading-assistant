package pgmemory

import "errors"

// RequiredSchemaVersion is the minimum migration version that the live memory
// runtime must observe before serving traffic.
const RequiredSchemaVersion = 21

var (
	ErrBankRequired             = errors.New("memory bank is required")
	ErrUnknownBank              = errors.New("memory bank is invalid")
	ErrInvalidMemoryItem        = errors.New("memory item validation failed")
	ErrReflectQueryRequired     = errors.New("memory reflection query is required")
	ErrMemoryItemIDRequired     = errors.New("memory item id is required")
	ErrMemoryItemNotFound       = errors.New("memory item not found")
	ErrDuplicateSourceReference = errors.New("memory duplicate source reference")
)
