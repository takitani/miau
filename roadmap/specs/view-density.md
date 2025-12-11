# UX-11: Compact/Comfortable View

## Overview
Allow users to choose email list density.

## Technical Requirements
```go
type ViewDensity string

const (
    DensityCompact     ViewDensity = "compact"     // Single line per email
    DensityComfortable ViewDensity = "comfortable" // Two lines
    DensitySpaciou     ViewDensity = "spacious"   // Three lines with preview
)

func (m Model) getEmailHeight() int {
    switch m.viewDensity {
    case DensityCompact:
        return 1
    case DensityComfortable:
        return 2
    case DensitySpaciou:
        return 3
    }
}
```

## Acceptance Criteria
- [ ] Three density options
- [ ] Persisted preference
- [ ] Adjusts list rendering

## Estimated Complexity
Low
