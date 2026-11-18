// Command adduser creates a user account. The password is typed at a hidden
// prompt (or piped on stdin), so it never appears in shell history.
//
//	go run ./services/adduser -org demo-org -email you@example.com -role admin
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"

	"github.com/S5in/mould-risk-iot/internal/apistore"
	"github.com/S5in/mould-risk-iot/internal/auth"
)

func main() {
	org := flag.String("org", "", "organisation slug, e.g. demo-org")
	email := flag.String("email", "", "login email")
	role := flag.String("role", "viewer", "admin or viewer")
	flag.Parse()
	if *org == "" || *email == "" {
		fail("usage: adduser -org SLUG -email EMAIL [-role admin|viewer]")
	}
	if *role != "admin" && *role != "viewer" {
		fail("role must be admin or viewer")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fail("DATABASE_URL is not set")
	}

	pw := readPassword("Password: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if again := readPassword("Repeat password: "); again != pw {
			fail("passwords do not match")
		}
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		fail(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fail(err.Error())
	}
	defer pool.Close()

	id, err := apistore.New(pool).CreateUser(ctx, *org, strings.TrimSpace(*email), hash, *role)
	if err != nil {
		fail("could not create user: " + err.Error())
	}
	fmt.Printf("created %s user %s in %s (id %s)\n", *role, *email, *org, id)
}

func readPassword(prompt string) string {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, prompt)
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fail(err.Error())
		}
		return string(b)
	}
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
