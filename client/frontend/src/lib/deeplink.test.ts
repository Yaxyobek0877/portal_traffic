// Vitest-style tests for the deeplink parser. Not currently wired into
// the test runner (we don't ship a frontend test harness yet) but kept
// alongside the implementation so the cases are documented and can be
// run when the test setup is added.
//
// To run manually: install vitest and `npm test`.

import { parseInvite } from "./deeplink";

function expect<T>(label: string, got: T, want: T) {
  const ok = JSON.stringify(got) === JSON.stringify(want);
  if (ok) {
    console.log(`  ✓ ${label}`);
  } else {
    console.error(`  ✗ ${label}\n    got:  ${JSON.stringify(got)}\n    want: ${JSON.stringify(want)}`);
  }
}

export function runDeeplinkTests() {
  console.log("parseInvite:");

  // Canonical join URL
  expect(
    "portal://join/<id>?code=<code>",
    parseInvite("portal://join/428591?code=739204"),
    { portalId: "428591", code: "739204" }
  );

  // PCP-1 alphanumeric code
  expect(
    "PCP-1 code",
    parseInvite("portal://join/428591?code=X3K9-A2F1"),
    { portalId: "428591", code: "X3K9-A2F1" }
  );

  // ID only
  expect(
    "id-only URL",
    parseInvite("portal://join/428591"),
    { portalId: "428591" }
  );

  // Legacy colon form
  expect(
    "portal://<id>:<code>",
    parseInvite("portal://428591:739204"),
    { portalId: "428591", code: "739204" }
  );

  // Plain text with ID + code
  expect(
    "Portal\\nID: 428591\\nCode: 739204",
    parseInvite("Portal\nID: 428591\nCode: 739204"),
    { portalId: "428591", code: "739204" }
  );

  // ID + code with whitespace separator only
  expect(
    "428591 739204",
    parseInvite("428591 739204"),
    { portalId: "428591", code: "739204" }
  );

  // Trailing slash
  expect(
    "trailing slash",
    parseInvite("portal://join/428591/"),
    { portalId: "428591" }
  );

  // Case-insensitive scheme
  expect(
    "uppercase scheme",
    parseInvite("PORTAL://join/428591?code=739204"),
    { portalId: "428591", code: "739204" }
  );

  // Garbage rejected
  expect("empty", parseInvite(""), null);
  expect("garbage", parseInvite("hello world"), null);

  // ID with bad chars rejected
  expect(
    "id with spaces",
    parseInvite("portal://join/has space?code=739204"),
    null
  );

  console.log("done");
}

// To run manually in a browser console:
//   import { runDeeplinkTests } from "./lib/deeplink.test";
//   runDeeplinkTests();
//
// We don't auto-run on import — vite tree-shakes this module out of
// the production bundle since nothing imports `runDeeplinkTests` in
// shipping code.
