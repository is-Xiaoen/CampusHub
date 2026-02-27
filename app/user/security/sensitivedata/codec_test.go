package sensitivedata

import "testing"

const (
	testAESKeyBase64  = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
	testHashKeyBase64 = "ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA="
)

func newTestCodec(t *testing.T) *Codec {
	t.Helper()

	codec, err := New(testAESKeyBase64, testHashKeyBase64)
	if err != nil {
		t.Fatalf("new codec failed: %v", err)
	}
	return codec
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	codec := newTestCodec(t)

	ciphertext, err := codec.Encrypt(fieldRealName, "张三")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if !codec.IsEncrypted(ciphertext) {
		t.Fatalf("ciphertext prefix missing: %s", ciphertext)
	}

	plaintext, err := codec.Decrypt(fieldRealName, ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if plaintext != "张三" {
		t.Fatalf("unexpected plaintext: %s", plaintext)
	}
}

func TestEncryptUsesRandomNonce(t *testing.T) {
	codec := newTestCodec(t)

	c1, err := codec.Encrypt(fieldStudentID, "20230001")
	if err != nil {
		t.Fatalf("encrypt #1 failed: %v", err)
	}
	c2, err := codec.Encrypt(fieldStudentID, "20230001")
	if err != nil {
		t.Fatalf("encrypt #2 failed: %v", err)
	}
	if c1 == c2 {
		t.Fatalf("ciphertext should be different for same plaintext")
	}
}

func TestDecryptRejectsPlaintext(t *testing.T) {
	codec := newTestCodec(t)

	_, err := codec.Decrypt(fieldStudentID, "20230001")
	if err == nil {
		t.Fatalf("decrypt plaintext should fail")
	}
}

func TestHashStudentID(t *testing.T) {
	codec := newTestCodec(t)

	h1, err := codec.HashStudentID("华中科技大学", "20230001")
	if err != nil {
		t.Fatalf("hash #1 failed: %v", err)
	}
	h2, err := codec.HashStudentID("华中科技大学", "20230001")
	if err != nil {
		t.Fatalf("hash #2 failed: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("same input must produce same hash")
	}
	if len(h1) != 64 {
		t.Fatalf("unexpected hash length: %d", len(h1))
	}

	h3, err := codec.HashStudentID("中山大学", "20230001")
	if err != nil {
		t.Fatalf("hash #3 failed: %v", err)
	}
	if h1 == h3 {
		t.Fatalf("different school should produce different hash")
	}
}
