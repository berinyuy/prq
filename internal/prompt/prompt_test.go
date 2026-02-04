package prompt

import "testing"

func TestRender(t *testing.T) {
	template := "User rules: {USER_RULES}\nRepo: {REPO}"
	snap := Snapshot{Repo: "octo/repo"}
	output := Render(template, []string{"Rule A"}, []string{}, snap)
	if output == template {
		t.Fatalf("expected replacements")
	}
}
