package harness

type MemoryTraceSink struct {
    Items []Trace
}

func (m *MemoryTraceSink) WriteTrace(t Trace) error {
    m.Items = append(m.Items, t)
    return nil
}
