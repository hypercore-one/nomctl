package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tyler-smith/go-bip39"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/wallet"
)

var znnCliWalletCreateNew = &cli.Command{
	Name:  "wallet.createNew",
	Usage: "passphrase [keyStoreName]",
	Action: func(cCtx *cli.Context) error {
		if !(cCtx.NArg() == 1 || cCtx.NArg() == 2) {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("wallet.createNew passphrase [keyStoreName]")
			return nil
		}

		// TODO finally implement a local keystore manager in go-zdk?
		entropy, _ := bip39.NewEntropy(256)
		mnemonic, _ := bip39.NewMnemonic(entropy)
		ks := &wallet.KeyStore{
			Entropy:  entropy,
			Seed:     bip39.NewSeed(mnemonic, ""),
			Mnemonic: mnemonic,
		}
		_, kp, _ := ks.DeriveForIndexPath(0)
		ks.BaseAddress = kp.Address

		name := ks.BaseAddress.String()
		if cCtx.NArg() == 2 {
			name = cCtx.Args().Get(1)
		}

		password := cCtx.Args().Get(0)
		kf, _ := ks.Encrypt(password)
		kf.Path = filepath.Join(walletDir, name)
		//kf.Write()
		// Uncomment when file mode is fixed
		keyFileJson, err := json.MarshalIndent(kf, "", "    ")
		if err != nil {
			return err
		}
		os.WriteFile(kf.Path, keyFileJson, 0600)

		fmt.Println("keyStore successfully created:", name)
		return nil
	},
}

var znnCliWalletCreateFromMnemonic = &cli.Command{
	Name:  "wallet.createFromMnemonic",
	Usage: "passphrase [keyStoreName]",
	Action: func(cCtx *cli.Context) error {
		if !(cCtx.NArg() == 2 || cCtx.NArg() == 3) {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("wallet.createFromMnemonic \"mnemonic\" passphrase [keyStoreName]")
			return nil
		}

		// TODO finally implement a local keystore manager in go-zdk?
		ms := cCtx.Args().Get(0)
		// TODO add in validation
		entropy, _ := bip39.EntropyFromMnemonic(ms)
		mnemonic, _ := bip39.NewMnemonic(entropy)
		ks := &wallet.KeyStore{
			Entropy:  entropy,
			Seed:     bip39.NewSeed(mnemonic, ""),
			Mnemonic: mnemonic,
		}
		_, kp, _ := ks.DeriveForIndexPath(0)
		ks.BaseAddress = kp.Address

		name := ks.BaseAddress.String()
		if cCtx.NArg() == 3 {
			name = cCtx.Args().Get(2)
		}

		password := cCtx.Args().Get(1)
		kf, _ := ks.Encrypt(password)
		kf.Path = filepath.Join(walletDir, name)
		//kf.Write()
		// Uncomment when file mode is fixed
		keyFileJson, err := json.MarshalIndent(kf, "", "    ")
		if err != nil {
			return err
		}
		os.WriteFile(kf.Path, keyFileJson, 0600)

		fmt.Println("keyStore successfully created:", name)
		return nil
	},
}

var znnCliWalletList = &cli.Command{
	Name:  "wallet.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("wallet.list")
			return nil
		}
		files, err := os.ReadDir(walletDir)
		if err != nil {
			return err
		}
		if len(files) > 0 {
			fmt.Println("Available keyStores:")
			for _, f := range files {
				if !f.IsDir() {
					fmt.Println(f.Name())
				}
			}
		} else {
			fmt.Println("No keyStores found")
		}
		return nil
	},
}
