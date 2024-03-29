package main

import (
	"encoding/json"
	"errors"
	"go/format"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moonstream-to/seer/evm"
	"github.com/moonstream-to/seer/starknet"
	"github.com/moonstream-to/seer/version"
)

func CreateRootCommand() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:   "seer",
		Short: "Seer: Generate interfaces and crawlers from various blockchains",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	completionCmd := CreateCompletionCommand(rootCmd)
	versionCmd := CreateVersionCommand()
	starknetCmd := CreateStarknetCommand()
	evmCmd := CreateEVMCommand()
	rootCmd.AddCommand(completionCmd, versionCmd, starknetCmd, evmCmd)

	// By default, cobra Command objects write to stderr. We have to forcibly set them to output to
	// stdout.
	rootCmd.SetOut(os.Stdout)

	return rootCmd
}

func CreateCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts for seer",
		Long: `Generate shell completion scripts for seer.

The command for each shell will print a completion script to stdout. You can source this script to get
completions in your current shell session. You can add this script to the completion directory for your
shell to get completions for all future sessions.

For example, to activate bash completions in your current shell:
		$ . <(seer completion bash)

To add seer completions for all bash sessions:
		$ seer completion bash > /etc/bash_completion.d/seer_completions`,
	}

	bashCompletionCmd := &cobra.Command{
		Use:   "bash",
		Short: "bash completions for seer",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(cmd.OutOrStdout())
		},
	}

	zshCompletionCmd := &cobra.Command{
		Use:   "zsh",
		Short: "zsh completions for seer",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenZshCompletion(cmd.OutOrStdout())
		},
	}

	fishCompletionCmd := &cobra.Command{
		Use:   "fish",
		Short: "fish completions for seer",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		},
	}

	powershellCompletionCmd := &cobra.Command{
		Use:   "powershell",
		Short: "powershell completions for seer",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
		},
	}

	completionCmd.AddCommand(bashCompletionCmd, zshCompletionCmd, fishCompletionCmd, powershellCompletionCmd)

	return completionCmd
}

func CreateVersionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of seer that you are currently using",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.SeerVersion)
		},
	}

	return versionCmd
}

func CreateStarknetCommand() *cobra.Command {
	starknetCmd := &cobra.Command{
		Use:   "starknet",
		Short: "Generate interfaces and crawlers for Starknet contracts",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	starknetABIParseCmd := CreateStarknetParseCommand()
	starknetABIGenGoCmd := CreateStarknetGenerateCommand()
	starknetCmd.AddCommand(starknetABIParseCmd, starknetABIGenGoCmd)

	return starknetCmd
}

func CreateStarknetParseCommand() *cobra.Command {
	var infile string
	var rawABI []byte
	var readErr error

	starknetParseCommand := &cobra.Command{
		Use:   "parse",
		Short: "Parse a Starknet contract's ABI and return seer's interal representation of that ABI",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if infile != "" {
				rawABI, readErr = os.ReadFile(infile)
			} else {
				rawABI, readErr = io.ReadAll(os.Stdin)
			}

			return readErr
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedABI, parseErr := starknet.ParseABI(rawABI)
			if parseErr != nil {
				return parseErr
			}

			content, marshalErr := json.Marshal(parsedABI)
			if marshalErr != nil {
				return marshalErr
			}

			cmd.Println(string(content))
			return nil
		},
	}

	starknetParseCommand.Flags().StringVarP(&infile, "abi", "a", "", "Path to contract ABI (default stdin)")

	return starknetParseCommand
}

func CreateStarknetGenerateCommand() *cobra.Command {
	var infile, packageName string
	var rawABI []byte
	var readErr error

	starknetGenerateCommand := &cobra.Command{
		Use:   "generate",
		Short: "Generate Go bindings for a Starknet contract from its ABI",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if infile != "" {
				rawABI, readErr = os.ReadFile(infile)
			} else {
				rawABI, readErr = io.ReadAll(os.Stdin)
			}

			return readErr
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			header, headerErr := starknet.GenerateHeader(packageName)
			if headerErr != nil {
				return headerErr
			}

			sections := []string{header}

			parsedABI, parseErr := starknet.ParseABI(rawABI)
			if parseErr != nil {
				return parseErr
			}

			code, codegenErr := starknet.Generate(parsedABI)
			if codegenErr != nil {
				return codegenErr
			}

			sections = append(sections, code)

			formattedCode, formattingErr := format.Source([]byte(strings.Join(sections, "\n\n")))
			if formattingErr != nil {
				return formattingErr
			}
			cmd.Println(string(formattedCode))
			return nil
		},
	}

	starknetGenerateCommand.Flags().StringVarP(&packageName, "package", "p", "", "The name of the package to generate")
	starknetGenerateCommand.Flags().StringVarP(&infile, "abi", "a", "", "Path to contract ABI (default stdin)")

	return starknetGenerateCommand
}

func CreateEVMCommand() *cobra.Command {
	evmCmd := &cobra.Command{
		Use:   "evm",
		Short: "Generate interfaces and crawlers for Ethereum Virtual Machine (EVM) contracts",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	evmGenerateCmd := CreateEVMGenerateCommand()
	evmCmd.AddCommand(evmGenerateCmd)

	return evmCmd
}

func CreateEVMGenerateCommand() *cobra.Command {
	var cli, noformat, includemain bool
	var infile, packageName, structName, bytecodefile, outfile, foundryBuildFile, hardhatBuildFile string
	var rawABI, bytecode []byte
	var readErr error
	var aliases map[string]string

	evmGenerateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Go bindings for an EVM contract from its ABI",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if packageName == "" {
				return errors.New("package name is required via --package/-p")
			}
			if structName == "" {
				return errors.New("struct name is required via --struct/-s")
			}

			if foundryBuildFile != "" {
				var contents []byte
				contents, readErr = os.ReadFile(foundryBuildFile)
				if readErr != nil {
					return readErr
				}

				type foundryBytecodeObject struct {
					Object string `json:"object"`
				}

				type foundryBuildArtifact struct {
					ABI      json.RawMessage       `json:"abi"`
					Bytecode foundryBytecodeObject `json:"bytecode"`
				}

				var artifact foundryBuildArtifact
				readErr = json.Unmarshal(contents, &artifact)
				rawABI = []byte(artifact.ABI)
				bytecode = []byte(artifact.Bytecode.Object)
			} else if hardhatBuildFile != "" {
				var contents []byte
				contents, readErr = os.ReadFile(hardhatBuildFile)
				if readErr != nil {
					return readErr
				}

				type hardhatBuildArtifact struct {
					ABI      json.RawMessage `json:"abi"`
					Bytecode string          `json:"bytecode"`
				}

				var artifact hardhatBuildArtifact
				readErr = json.Unmarshal(contents, &artifact)
				rawABI = []byte(artifact.ABI)
				bytecode = []byte(artifact.Bytecode)
			} else if infile != "" {
				rawABI, readErr = os.ReadFile(infile)
			} else {
				rawABI, readErr = io.ReadAll(os.Stdin)
			}

			if bytecodefile != "" {
				bytecode, readErr = os.ReadFile(bytecodefile)
			}

			return readErr
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			code, codeErr := evm.GenerateTypes(structName, rawABI, bytecode, packageName, aliases)
			if codeErr != nil {
				return codeErr
			}

			header, headerErr := evm.GenerateHeader(packageName, cli, includemain, foundryBuildFile, infile, bytecodefile, structName, outfile, noformat)
			if headerErr != nil {
				return headerErr
			}

			code = header + code

			if cli {
				code, readErr = evm.AddCLI(code, structName, noformat, includemain)
				if readErr != nil {
					return readErr
				}
			}

			if outfile != "" {
				writeErr := os.WriteFile(outfile, []byte(code), 0644)
				if writeErr != nil {
					return writeErr
				}
			} else {
				cmd.Println(code)
			}
			return nil
		},
	}

	evmGenerateCmd.Flags().StringVarP(&packageName, "package", "p", "", "The name of the package to generate")
	evmGenerateCmd.Flags().StringVarP(&structName, "struct", "s", "", "The name of the struct to generate")
	evmGenerateCmd.Flags().StringVarP(&infile, "abi", "a", "", "Path to contract ABI (default stdin)")
	evmGenerateCmd.Flags().StringVarP(&bytecodefile, "bytecode", "b", "", "Path to contract bytecode (default none - in this case, no deployment method is created)")
	evmGenerateCmd.Flags().BoolVarP(&cli, "cli", "c", false, "Add a CLI for interacting with the contract (default false)")
	evmGenerateCmd.Flags().BoolVar(&noformat, "noformat", false, "Set this flag if you do not want the generated code to be formatted (useful to debug errors)")
	evmGenerateCmd.Flags().BoolVar(&includemain, "includemain", false, "Set this flag if you want to generate a \"main\" function to execute the CLI and make the generated code self-contained - this option is ignored if --cli is not set")
	evmGenerateCmd.Flags().StringVarP(&outfile, "output", "o", "", "Path to output file (default stdout)")
	evmGenerateCmd.Flags().StringVar(&foundryBuildFile, "foundry", "", "If your contract is compiled using Foundry, you can specify a path to the build file here (typically \"<foundry project root>/out/<solidity filename>/<contract name>.json\") instead of specifying --abi and --bytecode separately")
	evmGenerateCmd.Flags().StringVar(&hardhatBuildFile, "hardhat", "", "If your contract is compiled using Hardhat, you can specify a path to the build file here (typically \"<path to solidity file in hardhat artifact directory>/<contract name>.json\") instead of specifying --abi and --bytecode separately")
	evmGenerateCmd.Flags().StringToStringVar(&aliases, "alias", nil, "A map of identifier aliases (e.g. --alias name=somename)")

	return evmGenerateCmd
}
