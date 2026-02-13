package cmd

import (
	"fmt"

	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/secrets"
	"github.com/spf13/cobra"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage encrypted secrets in the repository",
	}

	cmd.AddCommand(
		newSecretsInitCmd(),
		newSecretsEncryptCmd(),
		newSecretsDecryptCmd(),
		newSecretsStatusCmd(),
		newSecretsRotateCmd(),
	)

	return cmd
}

func newSecretsInitCmd() *cobra.Command {
	var identityPath string
	var importPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate or import an age identity for encrypting secrets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			id, err := secrets.Init(cfg.Repo.Path, secrets.InitOptions{
				IdentityPath: identityPath,
				ImportPath:   importPath,
				Force:        flagForce,
			})
			if err != nil {
				return err
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"status":       "ok",
					"private_path": id.PrivatePath,
					"public_key":   id.PublicKey,
					"imported":     importPath != "",
				})
			}

			if importPath != "" {
				out.Success("Imported age identity")
			} else {
				out.Success("Generated age identity")
			}
			out.Field("Private key", id.PrivatePath)
			out.Field("Public key", id.PublicKey)
			out.Info("")
			out.Info("Next steps:")
			out.Info("  1. Copy %s to your other machines", id.PrivatePath)
			out.Info("  2. Use 'dotctl secrets encrypt <file>' to protect sensitive files")
			out.Info("  3. Add 'decrypt: true' to manifest entries for encrypted files")

			return nil
		},
	}

	cmd.Flags().StringVar(&identityPath, "identity", "", "path for the identity file")
	cmd.Flags().StringVar(&importPath, "import", "", "import an existing identity file")

	return cmd
}

func newSecretsEncryptCmd() *cobra.Command {
	var recipientKey string
	var keep bool

	cmd := &cobra.Command{
		Use:   "encrypt <file> [file...]",
		Short: "Encrypt files for safe storage in the repository",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			type result struct {
				File      string `json:"file"`
				Encrypted string `json:"encrypted"`
				Error     string `json:"error,omitempty"`
			}

			var results []result
			var hasErr bool

			for _, file := range args {
				encPath, err := secrets.Encrypt(cfg.Repo.Path, file, secrets.EncryptOptions{
					RecipientKey: recipientKey,
					Keep:         keep,
				})
				if err != nil {
					results = append(results, result{File: file, Error: err.Error()})
					hasErr = true
					if !out.IsJSON() {
						out.Error("Failed to encrypt %s: %v", file, err)
					}
					continue
				}
				results = append(results, result{File: file, Encrypted: encPath})
				if !out.IsJSON() {
					out.Success("Encrypted: %s", encPath)
				}
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"results": results,
				})
			}

			if hasErr {
				return fmt.Errorf("some files failed to encrypt")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&recipientKey, "recipient", "", "age public key (default: from repo)")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep original plaintext file")

	return cmd
}

func newSecretsDecryptCmd() *cobra.Command {
	var identityPath string
	var keep bool
	var stdout bool

	cmd := &cobra.Command{
		Use:   "decrypt <file> [file...]",
		Short: "Decrypt encrypted files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			type result struct {
				File      string `json:"file"`
				Decrypted string `json:"decrypted,omitempty"`
				Error     string `json:"error,omitempty"`
			}

			var results []result
			var hasErr bool

			for _, file := range args {
				plaintext, decPath, err := secrets.Decrypt(cfg.Repo.Path, file, secrets.DecryptOptions{
					IdentityPath: identityPath,
					Keep:         keep || stdout,
					Stdout:       stdout,
				})
				if err != nil {
					results = append(results, result{File: file, Error: err.Error()})
					hasErr = true
					if !out.IsJSON() {
						out.Error("Failed to decrypt %s: %v", file, err)
					}
					continue
				}

				if stdout {
					fmt.Print(string(plaintext))
					continue
				}

				results = append(results, result{File: file, Decrypted: decPath})
				if !out.IsJSON() {
					out.Success("Decrypted: %s", decPath)
				}
			}

			if stdout {
				return nil
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"results": results,
				})
			}

			if hasErr {
				return fmt.Errorf("some files failed to decrypt")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&identityPath, "identity", "", "path to age identity file")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep encrypted file after decrypting")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "output decrypted content to stdout")

	return cmd
}

func newSecretsStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show secrets protection status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			status, err := secrets.GetStatus(cfg.Repo.Path, "")
			if err != nil {
				return err
			}

			if out.IsJSON() {
				return out.JSON(status)
			}

			// Identity info.
			if status.Identity != nil {
				out.Success("Identity: %s", status.Identity.PrivatePath)
				out.Field("Public key", status.Identity.PublicKey)
			} else {
				out.Warn("Identity: not configured (run 'dotctl secrets init')")
			}

			// Recipient info.
			if status.RecipientFile != "" {
				out.Success("Recipient: %s", status.RecipientFile)
			} else {
				out.Warn("Recipient: not found")
			}

			// Encrypted files.
			if len(status.EncryptedFiles) > 0 {
				out.Header("Protected files:")
				for _, f := range status.EncryptedFiles {
					out.Info("  %s", f.Path)
				}
			} else {
				out.Info("\nNo encrypted files found.")
			}

			// Unprotected files.
			if len(status.UnprotectedFiles) > 0 {
				out.Header("Unprotected sensitive files (warning):")
				for _, f := range status.UnprotectedFiles {
					out.Warn("  %s", f.Path)
				}
				out.Info("")
				out.Info("Run 'dotctl secrets encrypt <file>' to protect these files.")
			}

			return nil
		},
	}
}

func newSecretsRotateCmd() *cobra.Command {
	var identityPath string

	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Generate a new key and re-encrypt all protected files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			result, err := secrets.Rotate(cfg.Repo.Path, secrets.RotateOptions{
				IdentityPath: identityPath,
			})
			if err != nil {
				return err
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"status":          "ok",
					"public_key":      result.NewIdentity.PublicKey,
					"backup_key_path": result.BackupKeyPath,
					"re_encrypted":    result.ReEncrypted,
				})
			}

			out.Success("Generated new age identity")
			out.Field("Public key", result.NewIdentity.PublicKey)
			out.Field("Old key", result.BackupKeyPath)
			out.Info("")

			if len(result.ReEncrypted) > 0 {
				out.Info("Re-encrypted %d file(s):", len(result.ReEncrypted))
				for _, f := range result.ReEncrypted {
					out.Info("  %s", f)
				}
			}

			out.Info("")
			out.Info("Next steps:")
			out.Info("  1. Copy %s to your other machines", result.NewIdentity.PrivatePath)
			out.Info("  2. Run 'dotctl push' to sync re-encrypted files")
			out.Info("  3. Delete old key backup when confirmed: rm %s", result.BackupKeyPath)

			return nil
		},
	}

	cmd.Flags().StringVar(&identityPath, "identity", "", "path for the new identity file")

	return cmd
}
