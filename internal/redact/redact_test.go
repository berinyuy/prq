package redact

import "testing"

func TestRedactTokens(t *testing.T) {
	input := "token=ghp_1234567890abcdefghijklmnopqrstuvwxyz"
	output := Redact(input)
	if output == input {
		t.Fatalf("expected redaction")
	}
	if output == "" {
		t.Fatalf("expected output")
	}
}

func TestRedactJWT(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoiYWJjIn0.signature"
	output := Redact(input)
	if output == input {
		t.Fatalf("expected jwt redaction")
	}
}
