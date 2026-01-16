package contracts

type MemoryRetainRequest struct {
	Bank string     `json:"bank"`
	Item MemoryItem `json:"item"`
}

type MemoryRetainResponse struct {
	ID MemoryID `json:"id"`
}

type MemoryRecallRequest struct {
	Bank  string      `json:"bank"`
	Query MemoryQuery `json:"query"`
}

type MemoryRecallResponse struct {
	Items []MemoryItem `json:"items"`
}

type MemoryReflectRequest struct {
	Bank   string           `json:"bank"`
	Params ReflectionParams `json:"params"`
}

type MemoryReflectResponse struct {
	Items []MemoryItem `json:"items"`
}
