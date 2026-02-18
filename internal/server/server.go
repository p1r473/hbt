// Package server contains all logic related to the server.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lzambarda/hbt/internal/config"
)

func saveRoutines(ctx context.Context, graph Graph, cachePath string) {
	// Intercept termination signal to save the most recent knowledge
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-c:
		}

		if config.Debug {
			log.Println("Saving graph at", cachePath)
		}

		err := graph.Save(ctx, cachePath)
		if err != nil {
			log.Println(err)

			os.Exit(1)
		}

		os.Exit(0)
	}()

	// Periodically save the file
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			time.Sleep(config.SaveInterval)

			if config.Debug {
				log.Println("Saving graph at", cachePath)
			}

			err := graph.Save(ctx, cachePath)
			if err != nil {
				log.Println(err)

				os.Exit(1)
			}
		}
	}()
}

// Start the hbt server with the given graph and cache path.
func Start(ctx context.Context, graph Graph, cachePath string) error {
	saveRoutines(ctx, graph, cachePath)

	if config.Debug {
		log.Println("Starting server at", config.Port)
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp4", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", config.Port, err)
	}

	defer listener.Close() //nolint:errcheck // It is okay.

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx) //nolint:wrapcheck // No need.
		default:
		}

		connection, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}

		go handleConnection(ctx, connection, graph)
	}
}

// Stop the server.
func Stop() error {
	return nil
}

func handleConnection(ctx context.Context, connection net.Conn, graph Graph) {
	defer connection.Close() //nolint:errcheck // It is okay.

	buf := make([]byte, 0, 4096) //nolint:mnd // It's arbitrary

	tmp := make([]byte, 64) //nolint:mnd // It's arbitrary

	for {
		readBytes, err := connection.Read(tmp)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Error while reading request: %s", err)

				return
			}

			break
		}

		buf = append(buf, tmp[:readBytes]...)
	}

	args := strings.Split(string(buf), "\n")
	if config.Debug {
		log.Println(args)
	}

	result, err := ProcessCommand(ctx, args, graph)
	if err != nil {
		log.Printf("Error while processing command: %s", err)

		return
	}

	if result != "" {
		_, err = connection.Write([]byte(result))
		if err != nil {
			log.Printf("Error while writing response: %s", err)
		}
	}
}

const shrug = "¯\\_(ツ)_/¯"

// ProcessCommand processes the arguments and runs a command on the given Graph.
//
//nolint:gocyclo,cyclop,err113,mnd // It is okay.
func ProcessCommand(ctx context.Context, args []string, graph Graph) (string, error) {
	if len(args) == 0 {
		return "", errors.New("missing command")
	}

	switch args[0] {
	case "track":
		if len(args) != 4 {
			return "", fmt.Errorf("wrong number of arguments, expected 4, got %d", len(args))
		}

		userID, err := strconv.Atoi(args[1])
		if err != nil {
			return "", fmt.Errorf("invalid user ID: %w", err)
		}

		if args[3] == "" {
			return "", nil
		}

		if err := graph.Track(ctx, userID, args[2], args[3]); err != nil {
			log.Printf("Error tracking %v: %s\n", args[1:], err)
		}

	case "hint":
		if len(args) != 3 {
			return "", fmt.Errorf("wrong number of arguments, expected 3, got %d", len(args))
		}

		userID, err := strconv.Atoi(args[1])
		if err != nil {
			return "", fmt.Errorf("invalid user ID: %w", err)
		}

		// args[3] is unused for now
		hint, err := graph.Hint(ctx, userID, args[2])
		if err != nil {
			log.Printf("Error getting hint: %s\n", err)
		}

		if hint == "" {
			hint = shrug
		}

		return hint, nil
	case "end":
		if len(args) != 2 {
			return "", fmt.Errorf("wrong number of arguments, expected 2, got %d", len(args))
		}

		userID, err := strconv.Atoi(args[1])
		if err != nil {
			return "", fmt.Errorf("invalid user ID: %w", err)
		}

		if err := graph.End(ctx, userID); err != nil {
			log.Printf("Error ending session for %q: %s\n", args[1], err)
		}
	case "del":
		if len(args) != 4 {
			return "", fmt.Errorf("wrong number of arguments, expected 4, got %d", len(args))
		}

		userID, err := strconv.Atoi(args[1])
		if err != nil {
			return "", fmt.Errorf("invalid user ID: %w", err)
		}

		if err := graph.Delete(ctx, userID, args[2], args[3]); err != nil {
			log.Printf("Error deleting %v: %s\n", args[1:], err)
		}
	default:
		return "", fmt.Errorf("unknown command: %q", args[0])
	}

	return "", nil
}
