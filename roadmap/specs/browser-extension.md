# PL-07: Browser Extension

## Overview
Quick email access from browser toolbar.

## Features
- Unread count badge
- Quick compose popup
- Search from omnibox
- Gmail/Outlook redirect to miau

## Technical Requirements
- Chrome/Firefox manifest v3
- Connects to miau API server
- Offline badge caching

```javascript
// background.js
chrome.runtime.onInstalled.addListener(() => {
    // Poll for unread count
    setInterval(async () => {
        const count = await fetch(`${MIAU_API}/api/emails/unread/count`);
        chrome.action.setBadgeText({ text: count.toString() });
    }, 60000);
});
```

## Acceptance Criteria
- [ ] Shows unread count
- [ ] Quick compose works
- [ ] Search from omnibox
- [ ] Works with Chrome/Firefox

## Estimated Complexity
Medium
