# TH-14: Attachment Caching

## Overview
Cache downloaded attachments locally for offline access.

## Technical Requirements
```sql
CREATE TABLE attachment_cache (
    id INTEGER PRIMARY KEY,
    attachment_id INTEGER,
    local_path TEXT,
    size INTEGER,
    cached_at DATETIME,
    last_accessed DATETIME
);
```

```go
type AttachmentCache struct {
    basePath  string
    maxSize   int64
}

func (c *AttachmentCache) Get(attachmentID int64) (string, error) {
    // Check cache, return local path if exists
}

func (c *AttachmentCache) Store(attachmentID int64, data []byte) error {
    // Store to disk, update DB
}

func (c *AttachmentCache) Cleanup() {
    // LRU eviction when over maxSize
}
```

## Acceptance Criteria
- [ ] Attachments cached on download
- [ ] LRU eviction policy
- [ ] Configurable cache size
- [ ] Works offline

## Estimated Complexity
Low-Medium
