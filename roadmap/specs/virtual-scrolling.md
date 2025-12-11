# TH-09: Virtual Scrolling

## Overview

Only render visible emails for performance with large mailboxes.

## Technical Requirements

```go
type VirtualList struct {
    TotalItems     int
    VisibleItems   int
    ItemHeight     int
    ScrollOffset   int
    RenderBuffer   int  // Extra items above/below viewport
}

func (v *VirtualList) GetVisibleRange() (start, end int) {
    start = v.ScrollOffset / v.ItemHeight
    start = max(0, start - v.RenderBuffer)
    end = start + v.VisibleItems + (v.RenderBuffer * 2)
    end = min(v.TotalItems, end)
    return
}
```

## Acceptance Criteria

- [ ] Only visible items rendered
- [ ] Smooth scrolling maintained
- [ ] Memory usage constant regardless of list size
- [ ] Works with 10k+ emails

## Estimated Complexity

Medium
