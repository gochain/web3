package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/gochain/web3"
	"github.com/urfave/cli/v2"
)

func startCommand(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start a local GoChain development node via Docker",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "detach",
				Aliases: []string{"d"},
				Usage:   "Run container in background",
				Value:   true,
			},
			&cli.StringFlag{
				Name:  "env-file",
				Usage: "Path to custom environment configuration file",
			},
			pkFlag(false),
		},
		Action: func(c *cli.Context) error {
			return startLocalNode(ctx, c)
		},
	}
}

func startLocalNode(ctx context.Context, c *cli.Context) error {
	privateKey := c.String("private-key")
	var acc *web3.Account
	var err error
	if privateKey == "" {
		acc, err = web3.CreateAccount()
	} else {
		acc, err = web3.ParsePrivateKey(privateKey)
	}
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^/gochain$", "--format", "{{.Names}}")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker check failed: %w", err)
	}
	if len(stdoutStderr) != 0 {
		fmt.Println("Restarting existing container 'gochain'...")
		cmd = exec.CommandContext(ctx, "docker", "start", "gochain")
	} else {
		args := []string{
			"run", "-i",
			"--name", "gochain",
			"-p", "8545:8545",
			"-p", "8546:8546",
			"--entrypoint", "gochain",
		}
		if envFile := c.String("env-file"); envFile != "" {
			args = append(args, "--env-file", envFile)
		}
		if c.Bool("detach") {
			args = append(args, "-d")
		}
		args = append(args, "gochain/gochain", "--local", "--local.fund", acc.PublicKey())
		cmd = exec.CommandContext(ctx, "docker", args...)

		fmt.Println("Starting your own, personal GoChain instance...")
		fmt.Println(asciiLogo)
		fmt.Println()
		if privateKey == "" {
			fmt.Printf("We created an account for you to get started quickly.\n\nYour private key is:\n\n%s\n\n"+
				"Type: `export WEB3_PRIVATE_KEY=%s` to make using this tool easier.\n\n", acc.PrivateKey(), acc.PrivateKey())
		}
		fmt.Printf("Your account %s is pre-funded with 1000 GO.\n", acc.PublicKey())
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}
