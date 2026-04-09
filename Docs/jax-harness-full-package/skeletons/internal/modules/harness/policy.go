package harness

import "fmt"

type Mode string

const (
    ModeResearch Mode = "research"
    ModePaper    Mode = "paper"
    ModeLive     Mode = "live"
)

type Policy struct {
    Mode               Mode
    AdvisoryOnly       bool
    AllowExternalData  bool
    AllowPriceClaims   bool
    MaxToolCalls       int
    MaxSteps           int
}

func DefaultPolicy(mode Mode) Policy {
    return Policy{
        Mode:              mode,
        AdvisoryOnly:      true,
        AllowExternalData: false,
        AllowPriceClaims:  false,
        MaxToolCalls:      2,
        MaxSteps:          4,
    }
}

func (p Policy) CheckToolAllowed(def ToolDefinition) error {
    if !def.ReadOnly {
        return fmt.Errorf("tool %s is not read-only", def.Name)
    }
    return nil
}

func (p Policy) CheckAnswerAllowed(answer string) error {
    forbidden := []string{
        "I executed",
        "I approved",
        "I placed the trade",
    }
    for _, s := range forbidden {
        if containsInsensitive(answer, s) {
            return fmt.Errorf("forbidden action language detected: %s", s)
        }
    }
    return nil
}
