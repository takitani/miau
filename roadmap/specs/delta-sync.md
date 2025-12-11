# TH-13: Delta Sync

## Overview
Only sync changes since last sync instead of full folder scans.

## Technical Requirements
- Track last sync state per folder
- Use IMAP HIGHESTMODSEQ or CONDSTORE
- Fallback to UID comparison

```go
type DeltaSync struct {
    LastUID       uint32
    LastModSeq    uint64
    CheckpointAt  time.Time
}

func (s *SyncService) DeltaSync(ctx context.Context, folderID int64) error {
    state := s.getSyncState(folderID)
    // Fetch only UIDs > state.LastUID
    newUIDs, err := s.imap.GetUIDsGreaterThan(state.LastUID)
    // ...
}
```

## Acceptance Criteria
- [ ] Incremental sync working
- [ ] 90%+ reduction in sync time for unchanged folders
- [ ] Handles server changes correctly

## Estimated Complexity
Medium
