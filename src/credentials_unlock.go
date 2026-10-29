package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// promptUnlockCredentials asks for the vault master password on a TTY.
func promptUnlockCredentials(b *CredentialBroker) error {
	if b == nil || !b.Exists() || b.Unlocked() {
		return nil
	}
	if !isTerminal(0) {
		return fmt.Errorf("vault exists but no TTY for password prompt")
	}
	fmt.Fprint(os.Stderr, "Credential vault password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	return b.Unlock(string(pw))
}
