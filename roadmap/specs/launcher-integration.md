# PL-08: Raycast/Alfred Integration

## Overview
Quick email actions from macOS launchers.

## Raycast Extension
```typescript
export default function SearchEmails() {
    const { data } = useFetch<Email[]>(`${MIAU_API}/api/search?q=${query}`);
    return (
        <List>
            {data?.map(email => (
                <List.Item
                    key={email.id}
                    title={email.subject}
                    subtitle={email.from}
                    actions={
                        <ActionPanel>
                            <Action.Open title="Open in miau" target={`miau://email/${email.id}`} />
                            <Action title="Archive" onAction={() => archiveEmail(email.id)} />
                        </ActionPanel>
                    }
                />
            ))}
        </List>
    );
}
```

## Commands
- Search emails
- Compose new email
- View unread
- Quick archive

## Estimated Complexity
Low-Medium
