package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nuclear-ao3/internal/auth"
)

// runKeys runs "nuclear keys ...". It needs no database, so main calls it
// before opening one.
func runKeys(args []string) int {
	switch {
	case len(args) >= 1 && args[0] == "generate" && len(args) <= 2:
		path := filepath.Join("secrets", "signing-key-"+time.Now().Format("2006-01-02")+".pem")
		if len(args) == 2 {
			path = args[1]
		}
		return generateKey(path)
	case len(args) == 1 && args[0] == "show":
		return showKeys()
	default:
		fmt.Fprintln(os.Stderr, "usage: nuclear keys generate [path] | nuclear keys show")
		return 2
	}
}

func generateKey(path string) int {
	kid, err := auth.GenerateKeyFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "keys generate: %v\n", err)
		return 1
	}
	fmt.Printf(`Created a new signing key: %s
Key ID: %s

Do these now:

1. Save a copy of the file somewhere outside this server, such as a password
   manager or offline storage. It cannot be recovered from the server's
   backups: they deliberately do not contain it. If you lose every copy, the
   only fix is another new key, which logs everyone out.
2. Set SIGNING_KEY_FILE to the file's path as the app sees it (with Docker
   Compose that is /secrets/%s) and restart the app. (To sign the ID tokens
   given to other apps, make a second key and set OIDC_SIGNING_KEY_FILE.)
   On a hosting platform with no files, paste the whole file into the
   SIGNING_KEY (or OIDC_SIGNING_KEY) secret setting instead.

To change keys later: generate a new key, set SIGNING_KEY_FILE to it, and put
the old file in SIGNING_KEY_PREVIOUS_FILES so current logins keep working
(remove it after 14 days). If the old key may have leaked, leave
SIGNING_KEY_PREVIOUS_FILES empty instead. See docs/RUNBOOK.md.
`, path, kid, filepath.Base(path))
	return 0
}

// showKeys prints the key IDs the environment points at, so an admin can
// match a saved copy to what is running.
func showKeys() int {
	priv, prev, err := auth.LoadKeyFiles(os.Getenv("SIGNING_KEY_FILE"), os.Getenv("SIGNING_KEY_PREVIOUS_FILES"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "keys show: %v\n", err)
		if errors.Is(err, os.ErrNotExist) || os.Getenv("SIGNING_KEY_FILE") == "" {
			fmt.Fprintln(os.Stderr, "Create a key with: nuclear keys generate")
		}
		return 1
	}
	fmt.Println("current: ", auth.KeyID(&priv.PublicKey))
	for _, p := range prev {
		fmt.Println("previous:", auth.KeyID(p))
	}
	// The key that signs ID tokens for other apps, if it has its own file.
	if f := strings.TrimSpace(os.Getenv("OIDC_SIGNING_KEY_FILE")); f != "" {
		oidc, err := auth.LoadPrivateKeyFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "keys show: OIDC_SIGNING_KEY_FILE (%s): %v\n", f, err)
			return 1
		}
		fmt.Println("oidc:    ", auth.KeyID(&oidc.PublicKey))
	} else {
		fmt.Println("oidc:     (not set: ID tokens use the current key; a separate key is recommended)")
	}
	return 0
}
