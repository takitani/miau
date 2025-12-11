# UX-14: Notification Preferences

## Overview
Fine-grained control over email notifications.

## Technical Requirements
```yaml
notifications:
  enabled: true
  sound: true
  desktop: true
  filters:
    vip_only: false
    categories: []  # empty = all
    quiet_hours:
      enabled: true
      start: "22:00"
      end: "08:00"
  rules:
    - from: "boss@company.com"
      notify: always
    - from: "*@newsletter.com"
      notify: never
```

```go
type NotificationService interface {
    ShouldNotify(ctx context.Context, email *Email) bool
    Notify(ctx context.Context, title, body string) error
    SetQuietHours(start, end string) error
}
```

## Acceptance Criteria
- [ ] Per-sender rules
- [ ] Quiet hours
- [ ] VIP-only option
- [ ] Sound toggle

## Estimated Complexity
Low-Medium
