package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

// TestRunAuthListCommand covers the account listing. HOME is redirected to a
// temp dir so the real ~/.buzzrc is never touched.
func TestRunAuthListCommand(t *testing.T) {
	t.Run("lists the primary first", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := SaveConfig(&Config{Username: "alice", AuthToken: "a", Accounts: []Account{{Username: "bob", AuthToken: "b"}}}); err != nil {
			t.Fatal(err)
		}

		var out, errOut bytes.Buffer
		if code := runAuthListCommand(&out, &errOut); code != 0 {
			t.Fatalf("exit %d, stderr: %s", code, errOut.String())
		}
		if got := out.String(); got != "alice (primary)\nbob\n" {
			t.Errorf("got %q", got)
		}
		if strings.Contains(out.String(), "a") && strings.Contains(out.String(), "auth_token") {
			t.Error("tokens must never be printed")
		}
	})

	t.Run("errors when no config exists", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		var out, errOut bytes.Buffer
		if code := runAuthListCommand(&out, &errOut); code != 1 {
			t.Errorf("want exit 1, got %d", code)
		}
		if !strings.Contains(errOut.String(), "auth login") {
			t.Errorf("stderr should point at auth login, got %q", errOut.String())
		}
	})
}

func TestRunAuthLogoutCommand(t *testing.T) {
	t.Run("removes a secondary account", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := SaveConfig(&Config{Username: "alice", AuthToken: "a", Accounts: []Account{{Username: "bob", AuthToken: "b"}}}); err != nil {
			t.Fatal(err)
		}

		var out, errOut bytes.Buffer
		if code := runAuthLogoutCommand([]string{"bob"}, &out, &errOut); code != 0 {
			t.Fatalf("exit %d, stderr: %s", code, errOut.String())
		}
		config, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if config.Username != "alice" || len(config.Accounts) != 0 {
			t.Errorf("got %+v, want alice alone", config)
		}
	})

	t.Run("removing the primary promotes the next account", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := SaveConfig(&Config{Username: "alice", AuthToken: "a", Accounts: []Account{{Username: "bob", AuthToken: "b"}}}); err != nil {
			t.Fatal(err)
		}

		var out, errOut bytes.Buffer
		if code := runAuthLogoutCommand([]string{"alice"}, &out, &errOut); code != 0 {
			t.Fatalf("exit %d, stderr: %s", code, errOut.String())
		}
		config, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if config.Username != "bob" || config.AuthToken != "b" || len(config.Accounts) != 0 {
			t.Errorf("got %+v, want bob promoted to primary", config)
		}
	})

	t.Run("removing the last account leaves no credentials", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := SaveConfig(&Config{Username: "alice", AuthToken: "a"}); err != nil {
			t.Fatal(err)
		}

		var out, errOut bytes.Buffer
		if code := runAuthLogoutCommand([]string{"alice"}, &out, &errOut); code != 0 {
			t.Fatalf("exit %d, stderr: %s", code, errOut.String())
		}
		config, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if config.hasCredentials() {
			t.Errorf("got %+v, want no credentials left", config)
		}
	})

	t.Run("rejects an unknown account and bad arg counts", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := SaveConfig(&Config{Username: "alice", AuthToken: "a"}); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{{}, {"a", "b"}, {"carol"}} {
			var out, errOut bytes.Buffer
			if code := runAuthLogoutCommand(args, &out, &errOut); code != 1 {
				t.Errorf("args %v: want exit 1, got %d", args, code)
			}
		}
		// The config must be untouched by a failed logout.
		config, err := LoadConfig()
		if err != nil || config.Username != "alice" {
			t.Errorf("config should be unchanged, got %+v (%v)", config, err)
		}
	})
}

// TestReadCredentialsInputPiped covers the non-terminal path used by
// `buzz auth login < creds.json`: the whole stream is read, multiline JSON
// included, and nothing is treated as hidden terminal input.
func TestReadCredentialsInputPiped(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	const payload = "{\n  \"username\": \"alice\",\n  \"auth_token\": \"tok\"\n}\n"
	go func() {
		_, _ = w.WriteString(payload)
		_ = w.Close()
	}()
	got, hidden, err := readCredentialsInput(r)
	if err != nil {
		t.Fatal(err)
	}
	if hidden {
		t.Error("piped input must not be reported as hidden terminal input")
	}
	if got != payload {
		t.Errorf("got %q", got)
	}
	if u := credentialsUsername(got); u != "alice" {
		t.Errorf("username = %q", u)
	}
}

func TestInitialAuthModelMasksToken(t *testing.T) {
	if m := initialAuthModel(); m.textInput.EchoMode != textinput.EchoPassword {
		t.Errorf("EchoMode = %v, want EchoPassword", m.textInput.EchoMode)
	}
}

func TestCredentialsPrompt(t *testing.T) {
	if got := credentialsPrompt(true); got != "(input hidden) > " {
		t.Errorf("terminal prompt = %q", got)
	}
	if got := credentialsPrompt(false); got != "> " {
		t.Errorf("piped prompt = %q", got)
	}
}
