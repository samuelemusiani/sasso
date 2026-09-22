package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrCommandNotFound      = errors.New("command not found")
	ErrArgumentNotFound     = errors.New("argument not found")
	ErrCommandNotFoundInCLI = errors.New("command not found in CLI struct")
)

type Command struct {
	// Full name of the CLI command, such as `--config`
	full string
	// Short name such as '-c', but not always provided. If not provided empty string
	short string
	// If the argument takes an argument
	arg bool
	// Help string
	help string

	// After the parsing, this holds the argument. If the command does not take
	// an argument but the command is present the string will be empty
	parsedArg *string
}

// Parsed returns True if the command was parsed
func (c Command) Parsed() bool {
	return c.parsedArg != nil
}

func (c Command) Argument() string {
	if c.parsedArg == nil {
		panic("cannot get argument in cli if not parsed")
	}

	return *c.parsedArg
}

type Cli struct {
	// Program name
	name string
	// If the default program has an argument (like config path)
	arg bool
	// If the argument is parsed
	parsedArg *string

	// Help string to explain arg
	helps string

	commands []Command
}

func NewCli(name string, arg bool, help string) Cli {
	c := Cli{name: name, arg: arg, helps: help}
	c.AddCommand("--help", "-h", false, "Print this help message")

	return c
}

func (c *Cli) AddCommand(full string, short string, arg bool, help string) {
	c.commands = append(c.commands, Command{full: full, short: short, arg: arg, help: help})
}

func (c *Cli) Help() string {
	sb := strings.Builder{}

	sb.WriteString("Command: ")
	sb.WriteString(c.name)

	if c.arg {
		sb.WriteString(" <ARG>|<COMMAND>\n\nUsage:\n\t<ARG>\n\t\t")
		sb.WriteString(c.helps)
	}

	sb.WriteString("\nOptions:\n")

	for _, command := range c.commands {
		sb.WriteString("\t")
		sb.WriteString(command.full)

		if command.short != "" {
			sb.WriteString(", ")
			sb.WriteString(command.short)
		}

		sb.WriteString("\n\t\t")
		sb.WriteString(command.help)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (c *Cli) Parse(args []string) error {
	for i := 1; i < len(args); i++ {
		indx := slices.IndexFunc(c.commands, func(c Command) bool {
			return c.full == args[i] || (c.short != "" && c.short == args[i])
		})

		if indx == -1 {
			err := fmt.Errorf("%s: %w", args[i], ErrCommandNotFound)

			if !c.arg {
				return err
			}

			// we expect an argument for the main command
			// but it has to be the last one
			if i < len(args)-1 {
				return err
			}

			c.parsedArg = &args[i]

			return nil
		}

		// Command does not take an argument
		if !c.commands[indx].arg {
			s := ""
			c.commands[indx].parsedArg = &s

			continue
		}

		// The command takes an argument
		if i+1 >= len(args) {
			return fmt.Errorf("%s: %w", args[i], ErrArgumentNotFound)
		}

		i++
		c.commands[indx].parsedArg = &args[i]
	}

	return nil
}

func (c *Cli) GetCommand(name string) (Command, error) {
	for _, command := range c.commands {
		if command.full == name || (command.short != "" && command.short == name) {
			return command, nil
		}
	}

	return Command{}, ErrCommandNotFoundInCLI
}

func (c *Cli) MustGetCommand(name string) Command {
	command, err := c.GetCommand(name)
	if err != nil {
		panic(err)
	}

	return command
}

func (c *Cli) ArgWasParsed() bool {
	return c.parsedArg != nil
}

func (c *Cli) Argument() string {
	if c.parsedArg == nil {
		panic("CLI argument is nil")
	}

	return *c.parsedArg
}
