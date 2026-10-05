# apex-claude

## Apex Agents

- **Start:** invoke your Agent skill, `/<agent>`.
- **Sign off:** say "sign off" or run `/sign-off`.

### Cross-Agent communication

1. Talk to a peer Agent whenever the work needs it: `ListAgents` finds it, `SendMessage` reaches it.
2. When a peer asks you for something: with no item in progress, act on it and reply right away; with an item in
   progress, file the request in your own ledger and reply with the new item's id.
3. Keep peer traffic out of the operator's chat. Acknowledge a peer's message in one line and answer the peer; never
   repeat to the operator what you sent to a peer. Bring it to the operator only when you need the operator's answer.
