package guard

import (
	"testing"
)

func TestGuardCheckBlastRadius(t *testing.T) {
	alert := CheckBlastRadius("rm", []string{"-rf", "*"})
	if alert == nil || !alert.Triggered {
		t.Fatalf("expected rm -rf * to trigger guard")
	}

	alert = CheckBlastRadius("git", []string{"reset", "--hard"})
	if alert == nil || !alert.Triggered {
		t.Fatalf("expected git reset --hard to trigger guard")
	}

	alert = CheckBlastRadius("git", []string{"push", "--force", "origin", "main"})
	if alert == nil || !alert.Triggered {
		t.Fatalf("expected git push --force to trigger guard")
	}

	alert = CheckBlastRadius("ls", []string{"-la"})
	if alert != nil {
		t.Fatalf("expected ls -la NOT to trigger guard")
	}
}
