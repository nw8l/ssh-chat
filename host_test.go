package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"strings"
	"testing"

	"github.com/shazow/ssh-chat/chat"
	"github.com/shazow/ssh-chat/sshd"
	"golang.org/x/crypto/ssh"
)

func stripPrompt(s string) string {
	pos := strings.LastIndex(s, "\033[K")
	if pos < 0 {
		return s
	}
	return s[pos+3:]
}

func TestHostGetPrompt(t *testing.T) {
	var expected, actual string

	u := chat.NewUser("foo")
	u.SetColorIdx(2)

	actual = GetPrompt(u)
	expected = "[foo] "
	if actual != expected {
		t.Errorf("Got: %q; Expected: %q", actual, expected)
	}

	u.Config.Theme = &chat.Themes[0]
	actual = GetPrompt(u)
	expected = "[\033[38;05;2mfoo\033[0m] "
	if actual != expected {
		t.Errorf("Got: %q; Expected: %q", actual, expected)
	}
}

func TestHostNameCollision(t *testing.T) {
	key, err := sshd.NewRandomSigner(512)
	if err != nil {
		t.Fatal(err)
	}
	config := sshd.MakeNoAuth()
	config.AddHostKey(key)

	s, err := sshd.ListenSSH(":0", config)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	host := NewHost(s)
	go host.Serve()

	done := make(chan struct{}, 1)

	// First client
	go func() {
		err = sshd.NewClientSession(s.Addr().String(), "foo", func(r io.Reader, w io.WriteCloser) {
			scanner := bufio.NewScanner(r)

			// Consume the initial buffer
			scanner.Scan()
			actual := scanner.Text()
			if !strings.HasPrefix(actual, "[foo] ") {
				t.Errorf("First client failed to get 'foo' name.")
			}

			actual = stripPrompt(actual)
			expected := " * foo joined. (Connected: 1)"
			if actual != expected {
				t.Errorf("Got %q; expected %q", actual, expected)
			}

			// Ready for second client
			done <- struct{}{}

			scanner.Scan()
			actual = stripPrompt(scanner.Text())
			expected = " * Guest1 joined. (Connected: 2)"
			if actual != expected {
				t.Errorf("Got %q; expected %q", actual, expected)
			}

			// Wrap it up.
			close(done)
		})
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Wait for first client
	<-done

	// Second client
	err = sshd.NewClientSession(s.Addr().String(), "foo", func(r io.Reader, w io.WriteCloser) {
		scanner := bufio.NewScanner(r)

		// Consume the initial buffer
		scanner.Scan()
		actual := scanner.Text()
		if !strings.HasPrefix(actual, "[Guest1] ") {
			t.Errorf("Second client did not get Guest1 name.")
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	<-done
}

func TestHostWhitelist(t *testing.T) {
	key, err := sshd.NewRandomSigner(512)
	if err != nil {
		t.Fatal(err)
	}

	auth := NewAuth()
	config := sshd.MakeAuth(auth)
	config.AddHostKey(key)

	s, err := sshd.ListenSSH(":0", config)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	host := NewHost(s)
	host.auth = auth
	go host.Serve()

	target := s.Addr().String()

	err = sshd.NewClientSession(target, "foo", func(r io.Reader, w io.WriteCloser) {})
	if err != nil {
		t.Error(err)
	}

	clientkey, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}

	clientpubkey, _ := ssh.NewPublicKey(clientkey.Public())
	auth.Whitelist(clientpubkey)

	err = sshd.NewClientSession(target, "foo", func(r io.Reader, w io.WriteCloser) {})
	if err == nil {
		t.Error("Failed to block unwhitelisted connection.")
	}
}