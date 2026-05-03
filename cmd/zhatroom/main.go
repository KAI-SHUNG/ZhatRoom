package main

import (
	"ZhatRoom/internal/server"
	"bufio"
	"fmt"
	"os"
	"strings"
)

const authorizedKeysPath = "/opt/zhatroom/authorized_keys"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: zhatroom <user add|user list|user remove> [args]\n")
		os.Exit(1)
	}

	db := server.InitDB()

	switch os.Args[1] {
	case "user":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: zhatroom user <add|list|remove> [username]\n")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "add":
			if len(os.Args) < 4 {
				fmt.Fprintf(os.Stderr, "Usage: zhatroom user add <username> < pubkey\n")
				os.Exit(1)
			}
			cmdUserAdd(db, os.Args[3])
		case "list":
			cmdUserList(db)
		case "remove":
			if len(os.Args) < 4 {
				fmt.Fprintf(os.Stderr, "Usage: zhatroom user remove <username>\n")
				os.Exit(1)
			}
			cmdUserRemove(db, os.Args[3])
		default:
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", os.Args[2])
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func cmdUserAdd(db *server.Storage, username string) {
	exists, err := db.UserExists(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB error: %v\n", err)
		os.Exit(1)
	}
	if exists {
		fmt.Fprintf(os.Stderr, "User %s already exists\n", username)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintf(os.Stderr, "Error: no public key provided on stdin\n")
		os.Exit(1)
	}
	pubkey := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(pubkey, "ssh-") {
		fmt.Fprintf(os.Stderr, "Error: invalid public key (must start with 'ssh-')\n")
		os.Exit(1)
	}

	if err := db.NewUser(username, username); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create user %s: %v\n", username, err)
		os.Exit(1)
	}

	line := fmt.Sprintf(`command="/opt/zhatroom/entrypoint.sh %s",restrict %s`, username, pubkey)
	f, err := os.OpenFile(authorizedKeysPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open authorized_keys: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write authorized_keys: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %s added successfully\n", username)
}

func cmdUserList(db *server.Storage) {
	users, err := db.ListUsers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB error: %v\n", err)
		os.Exit(1)
	}
	if len(users) == 0 {
		fmt.Println("No users found")
		return
	}
	for _, u := range users {
		fmt.Printf("  %s  (%s)\n", u.ID, u.Nickname)
	}
}

func cmdUserRemove(db *server.Storage, username string) {
	exists, err := db.UserExists(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB error: %v\n", err)
		os.Exit(1)
	}
	if !exists {
		fmt.Fprintf(os.Stderr, "User %s does not exist\n", username)
		os.Exit(1)
	}

	if err := db.DeleteUser(username); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete user %s: %v\n", username, err)
		os.Exit(1)
	}

	if err := removeKeyFromFile(authorizedKeysPath, username); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: user deleted from DB but failed to update authorized_keys: %v\n", err)
	}

	fmt.Printf("User %s removed successfully\n", username)
}

func removeKeyFromFile(path, username string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	pattern := fmt.Sprintf("entrypoint.sh %s\"", username)
	var out strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.Contains(line, pattern) {
			out.WriteString(line + "\n")
		}
	}

	return os.WriteFile(path, []byte(out.String()), 0600)
}
