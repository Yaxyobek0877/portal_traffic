package signaling

// DefaultURL is the production signaling endpoint that ships with the
// project. The desktop app (Phase 3+) uses this as its default; users
// can override it from the Settings → Network page or via the
// PORTAL_SIGNALING_URL environment variable.
//
// If you fork the project to run your own infrastructure, change this
// one line. Keeping it a single string constant means a fork doesn't
// have to track changes anywhere else.
const DefaultURL = "wss://signaling.1pro.uz/ws"
