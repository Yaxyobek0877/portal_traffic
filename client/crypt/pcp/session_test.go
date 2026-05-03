package pcp

import (
	"bytes"
	"testing"
)

// pairSetup builds two PairSessions matching what Alice and Bob would
// each derive given a successful X25519 handshake.
func pairSetup(t *testing.T) (alice, bob *PairSession) {
	t.Helper()

	// Each peer has its own long-term identity.
	idA, _ := NewIdentity()
	idB, _ := NewIdentity()

	// Each peer generates an ephemeral session keypair.
	ephA, _ := GenerateX25519()
	ephB, _ := GenerateX25519()

	// Both compute the same shared secret.
	sharedA, err := SharedSecret(ephA.Private(), ephB.Public)
	if err != nil {
		t.Fatal(err)
	}
	sharedB, err := SharedSecret(ephB.Private(), ephA.Public)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sharedA, sharedB) {
		t.Fatal("X25519 shared secrets differ")
	}

	portalID := "428591"

	alice, err = DeriveSession(sharedA, portalID, idA.Public, idB.Public)
	if err != nil {
		t.Fatal(err)
	}
	bob, err = DeriveSession(sharedB, portalID, idB.Public, idA.Public)
	if err != nil {
		t.Fatal(err)
	}
	return alice, bob
}

func TestSession_AliceToBob(t *testing.T) {
	alice, bob := pairSetup(t)

	for i, plaintext := range [][]byte{
		[]byte("hello bob"),
		[]byte(""),
		bytes.Repeat([]byte{0x41}, 100_000),
	} {
		frame, err := alice.Seal(plaintext)
		if err != nil {
			t.Fatalf("case %d: Alice.Seal: %v", i, err)
		}
		opened, err := bob.Open(frame)
		if err != nil {
			t.Fatalf("case %d: Bob.Open: %v", i, err)
		}
		if !bytes.Equal(opened, plaintext) {
			t.Fatalf("case %d: Bob got %q, want %q", i, opened, plaintext)
		}
	}
}

func TestSession_BobToAlice(t *testing.T) {
	alice, bob := pairSetup(t)

	frame, err := bob.Seal([]byte("response from bob"))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := alice.Open(frame)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened, []byte("response from bob")) {
		t.Fatalf("alice opened to %q, want %q", opened, "response from bob")
	}
}

func TestSession_BidirectionalInterleaved(t *testing.T) {
	alice, bob := pairSetup(t)

	for i := 0; i < 10; i++ {
		// Alice → Bob
		out := []byte{byte(i)}
		f, _ := alice.Seal(out)
		got, err := bob.Open(f)
		if err != nil {
			t.Fatalf("alice→bob iter %d: %v", i, err)
		}
		if !bytes.Equal(got, out) {
			t.Fatal("alice→bob mismatch")
		}
		// Bob → Alice
		out2 := []byte{byte(100 + i)}
		f2, _ := bob.Seal(out2)
		got2, err := alice.Open(f2)
		if err != nil {
			t.Fatalf("bob→alice iter %d: %v", i, err)
		}
		if !bytes.Equal(got2, out2) {
			t.Fatal("bob→alice mismatch")
		}
	}
}

func TestSession_CrossKeyFails(t *testing.T) {
	alice, bob := pairSetup(t)

	// A frame Alice sealed should NOT open with Alice's own session
	// (her own k_recv is for traffic FROM Bob, not from herself).
	f, _ := alice.Seal([]byte("alice's msg"))
	if _, err := alice.Open(f); err == nil {
		t.Fatal("Alice opened her own frame — direction keys not separating traffic")
	}
	// Bob's frame should similarly fail to open with Bob's session.
	f2, _ := bob.Seal([]byte("bob's msg"))
	if _, err := bob.Open(f2); err == nil {
		t.Fatal("Bob opened his own frame")
	}
}

func TestSession_DifferentPortalIDsDontCollide(t *testing.T) {
	idA, _ := NewIdentity()
	idB, _ := NewIdentity()
	ephA, _ := GenerateX25519()
	ephB, _ := GenerateX25519()
	shared, _ := SharedSecret(ephA.Private(), ephB.Public)

	a1, _ := DeriveSession(shared, "111111", idA.Public, idB.Public)
	a2, _ := DeriveSession(shared, "222222", idA.Public, idB.Public)

	frame, _ := a1.Seal([]byte("portal 111111"))
	// Try to open with the other portal's session — should fail.
	if _, err := a2.Open(frame); err == nil {
		t.Fatal("session for portal 222222 opened a frame from portal 111111")
	}
}

func TestSession_ReplayDetection(t *testing.T) {
	alice, bob := pairSetup(t)

	frame, _ := alice.Seal([]byte("original"))
	_, err := bob.Open(frame)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}
	// Send the SAME frame again — replay must be rejected.
	if _, err := bob.Open(frame); err == nil {
		t.Fatal("replay accepted")
	}
}

func TestSession_OutOfOrderRejected(t *testing.T) {
	alice, bob := pairSetup(t)
	f1, _ := alice.Seal([]byte("first"))
	f2, _ := alice.Seal([]byte("second"))

	// Receive f2 first (skip ahead).
	if _, err := bob.Open(f2); err != nil {
		t.Fatalf("Bob.Open(f2): %v", err)
	}
	// Now f1 should be rejected as out-of-order/replay.
	if _, err := bob.Open(f1); err == nil {
		t.Fatal("out-of-order f1 accepted after f2")
	}
}

func TestSession_TamperDetection(t *testing.T) {
	alice, bob := pairSetup(t)
	frame, _ := alice.Seal([]byte("important"))

	tampered := append([]byte{}, frame...)
	tampered[len(tampered)/2] ^= 0x01
	if _, err := bob.Open(tampered); err == nil {
		t.Fatal("tampered frame accepted")
	}
}

func TestSession_RejectShortFrame(t *testing.T) {
	_, bob := pairSetup(t)
	for _, n := range []int{0, 1, 8, 16, 23} {
		short := make([]byte, n)
		if _, err := bob.Open(short); err == nil {
			t.Errorf("Open accepted %d-byte input", n)
		}
	}
}

func TestSession_CounterAdvances(t *testing.T) {
	alice, _ := pairSetup(t)
	if alice.SendCounter() != 0 {
		t.Fatalf("initial counter = %d, want 0", alice.SendCounter())
	}
	for i := 0; i < 5; i++ {
		_, err := alice.Seal([]byte("x"))
		if err != nil {
			t.Fatal(err)
		}
	}
	if alice.SendCounter() != 5 {
		t.Fatalf("counter after 5 seals = %d, want 5", alice.SendCounter())
	}
}
