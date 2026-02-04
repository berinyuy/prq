package github

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

type recordingRunner struct {
	Args  []string
	Stdin []byte

	Output []byte
}

func (r *recordingRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	_ = ctx
	r.Args = append([]string(nil), args...)
	r.Stdin = append([]byte(nil), stdin...)
	return r.Output, nil
}

func TestCreateReview(t *testing.T) {
	runner := &recordingRunner{Output: []byte(`{"id":123,"html_url":"https://example.test/review/123"}`)}
	client := NewClient(runner)

	req := CreateReviewRequest{
		Body:  "Hello",
		Event: "COMMENT",
		Comments: []ReviewComment{
			{Path: "a.txt", Position: 3, Body: "Inline"},
		},
	}
	resp, err := client.CreateReview(context.Background(), "acme/app", 42, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != 123 {
		t.Fatalf("expected ID=123, got %d", resp.ID)
	}

	expectedArgs := []string{"api", "-X", "POST", "repos/acme/app/pulls/42/reviews", "--input", "-"}
	if !reflect.DeepEqual(runner.Args, expectedArgs) {
		t.Fatalf("unexpected args: %#v", runner.Args)
	}

	var got CreateReviewRequest
	if err := json.Unmarshal(runner.Stdin, &got); err != nil {
		t.Fatalf("stdin not valid json: %v", err)
	}
	if got.Event != "COMMENT" || got.Body != "Hello" || len(got.Comments) != 1 {
		t.Fatalf("unexpected request payload: %#v", got)
	}
}
