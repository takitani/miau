# PL-06: Mobile PWA

## Overview

Create a Progressive Web App version of miau optimized for mobile devices.

## User Stories

1. As a user, I want to access email on my phone
2. As a user, I want to install miau on my home screen
3. As a user, I want push notifications for new emails
4. As a user, I want offline access to recent emails

## Technical Requirements

### PWA Setup

```json
// manifest.json
{
  "name": "miau - Email Client",
  "short_name": "miau",
  "description": "Local-first email client with AI",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#1a1a1a",
  "theme_color": "#4ECDC4",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

### Service Worker

```javascript
// sw.js
const CACHE_NAME = 'miau-v1';
const OFFLINE_URLS = [
  '/',
  '/offline.html',
  '/static/css/app.css',
  '/static/js/app.js',
  '/static/icons/icon-192.png'
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => {
      return cache.addAll(OFFLINE_URLS);
    })
  );
});

self.addEventListener('fetch', event => {
  event.respondWith(
    caches.match(event.request).then(response => {
      // Return cached version or fetch
      return response || fetch(event.request).then(fetchResponse => {
        // Cache API responses for offline
        if (event.request.url.includes('/api/emails')) {
          caches.open(CACHE_NAME).then(cache => {
            cache.put(event.request, fetchResponse.clone());
          });
        }
        return fetchResponse;
      });
    }).catch(() => {
      // Offline fallback
      return caches.match('/offline.html');
    })
  );
});

// Push notifications
self.addEventListener('push', event => {
  const data = event.data.json();
  self.registration.showNotification(data.title, {
    body: data.body,
    icon: '/icons/icon-192.png',
    badge: '/icons/badge.png',
    data: data.url
  });
});
```

### Mobile-Optimized UI

```html
<!-- Mobile layout -->
<div class="mobile-app">
  <!-- Bottom navigation -->
  <nav class="bottom-nav">
    <a href="/" class="nav-item active">
      <span class="icon">📥</span>
      <span class="label">Inbox</span>
      <span class="badge">3</span>
    </a>
    <a href="/search" class="nav-item">
      <span class="icon">🔍</span>
      <span class="label">Search</span>
    </a>
    <a href="/compose" class="nav-item">
      <span class="icon">✏️</span>
      <span class="label">Compose</span>
    </a>
    <a href="/settings" class="nav-item">
      <span class="icon">⚙️</span>
      <span class="label">Settings</span>
    </a>
  </nav>

  <!-- Pull to refresh -->
  <div class="email-list" data-pull-refresh>
    <!-- Emails -->
  </div>
</div>
```

### Touch Gestures

```javascript
// Swipe actions
const emailList = document.querySelector('.email-list');

emailList.addEventListener('touchstart', handleTouchStart);
emailList.addEventListener('touchmove', handleTouchMove);
emailList.addEventListener('touchend', handleTouchEnd);

function handleSwipe(element, direction) {
  if (direction === 'left') {
    // Archive
    archiveEmail(element.dataset.id);
    element.classList.add('swipe-left');
  } else if (direction === 'right') {
    // Delete
    deleteEmail(element.dataset.id);
    element.classList.add('swipe-right');
  }
}
```

### Push Notifications

```go
// Server-side push
func (s *Server) sendPushNotification(subscription PushSubscription, email *Email) error {
    payload := map[string]string{
        "title": fmt.Sprintf("New email from %s", email.FromName),
        "body":  email.Subject,
        "url":   fmt.Sprintf("/email/%d", email.ID),
    }

    return webpush.SendNotification(subscription, payload)
}

// Register subscription endpoint
func (s *Server) registerPush(w http.ResponseWriter, r *http.Request) {
    var sub PushSubscription
    json.NewDecoder(r.Body).Decode(&sub)

    s.storage.SavePushSubscription(r.Context(), sub)

    w.WriteHeader(201)
}
```

## Mobile-Specific Features

### Email List

```
┌────────────────────────────────┐
│ miau                 🔍 ⋮     │
├────────────────────────────────┤
│ ↓ Pull to refresh              │
├────────────────────────────────┤
│ ●  John Smith                  │
│    Project Update              │
│    10:30 AM                    │
│ ← swipe left: archive          │
│ → swipe right: delete          │
├────────────────────────────────┤
│    Newsletter                  │
│    Weekly Digest               │
│    Yesterday                   │
├────────────────────────────────┤
│    ...                         │
├────────────────────────────────┤
│ 📥  🔍  ✏️  ⚙️              │
│Inbox Search Compose Settings   │
└────────────────────────────────┘
```

### Email View

```
┌────────────────────────────────┐
│ ←  Project Update         ⋮   │
├────────────────────────────────┤
│ John Smith                     │
│ john@example.com               │
│ To: me                         │
│ Dec 15, 10:30 AM               │
├────────────────────────────────┤
│                                │
│ Hi,                            │
│                                │
│ Here's the project update...   │
│                                │
│ Best regards,                  │
│ John                           │
│                                │
├────────────────────────────────┤
│ [Reply] [Forward] [Archive]    │
└────────────────────────────────┘
```

## Testing

1. Test PWA installation
2. Test offline functionality
3. Test push notifications
4. Test swipe gestures
5. Test on various devices
6. Test responsive breakpoints

## Acceptance Criteria

- [ ] Can install as PWA
- [ ] Works offline (cached emails)
- [ ] Push notifications work
- [ ] Swipe gestures work
- [ ] Pull-to-refresh works
- [ ] Responsive on all screen sizes
- [ ] Native-like experience

## Configuration

```yaml
# config.yaml
pwa:
  enabled: true
  push_notifications: true
  vapid_public_key: "${VAPID_PUBLIC_KEY}"
  vapid_private_key: "${VAPID_PRIVATE_KEY}"
```

## Estimated Complexity

Medium-High - PWA features plus mobile optimization
