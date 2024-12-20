package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	signer "github.com/hypercore-one/go-zdk/wallet"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/wallet"
	"golang.org/x/term"
)

var (
	utilZ = types.ParseZTSPanic("zts1utylzxxxxxxxxxxx6agxt0")
	utilQ = types.ParseZTSPanic("zts1utylqxxxxxxxxxxxdzq2gc")
)

func areValidPageVars(pageIndex int, pageSize int) bool {
	if pageIndex < 0 {
		fmt.Println("Error! The page index must be a positive integer")
		return false
	}
	if pageSize < 1 || pageSize > rpcMaxPageSize {
		fmt.Println("Error! The page size must be greater than 0 and less than or equal to", rpcMaxPageSize)
		return false
	}
	return true
}

func getTokenStandard(zts string) (types.ZenonTokenStandard, error) {
	l := strings.ToLower(zts)
	if l == "znn" {
		return types.ZnnTokenStandard, nil
	} else if l == "qsr" {
		return types.QsrTokenStandard, nil
	} else if l == "utilz" {
		return utilZ, nil
	} else if l == "utilq" {
		return utilQ, nil
	} else {
		t, err := types.ParseZTS(l)
		if err != nil {
			return types.ZeroTokenStandard, err
		}
		return t, nil
	}

}

func getZnnCliSigner(walletDir string, cCtx *cli.Context) (signer.Signer, error) {

	var keyStorePath string

	// TODO use go-zdk keystore manager when available
	files, err := os.ReadDir(walletDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		fmt.Println("Error! No keystore in the default directory")
		os.Exit(1)

	} else if cCtx.IsSet("keyStore") {
		keyStorePath = filepath.Join(walletDir, cCtx.String("keyStore"))
		info, err := os.Stat(keyStorePath)
		if os.IsNotExist(err) || info.IsDir() {
			fmt.Println("Error! The keyStore", cCtx.String("keyStore"), "does not exist in the default directory")
			os.Exit(1)
		}
	} else if len(files) == 1 {
		fmt.Println("Using the default keyStore", files[0].Name())
		keyStorePath = filepath.Join(walletDir, files[0].Name())
	} else {
		fmt.Println("Error! Please provide a keyStore or an address. Use 'wallet.list' to list all available keyStores")
		os.Exit(1)
	}

	var passphrase string
	if !cCtx.IsSet("passphrase") {
		fmt.Println("Insert passphrase:")
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		passphrase = string(pw)
		if err != nil {
			return nil, err
		}
	} else {
		passphrase = cCtx.String("passphrase")
	}

	kf, err := wallet.ReadKeyFile(keyStorePath)
	if err != nil {
		return nil, err
	}
	ks, err := kf.Decrypt(passphrase)
	if err != nil {
		if err == wallet.ErrWrongPassword {
			fmt.Println("Error! Invalid passphrase for keyStore", cCtx.String("keyStore"))
			os.Exit(1)
		}
		return nil, err
	}

	_, keyPair, err := ks.DeriveForIndexPath(uint32(cCtx.Int("index")))
	if err != nil {
		return nil, err
	}
	kp := signer.NewSigner(keyPair)

	return kp, nil

}

var znnCliSubcommands = []*cli.Command{
	znnCliSend,
	znnCliReceiveAll,
	znnCliUnreceived,
	znnCliBalance,
	znnCliFrontierMomentum,
	znnCliWalletCreateNew,
	znnCliWalletCreateFromMnemonic,
	znnCliWalletList,
	//		znnCliWalletDeriveAddresses,
	znnCliPlasmaList,
	znnCliPlasmaGet,
	znnCliPlasmaFuse,
	znnCliPlasmaCancel,
	znnCliPillarList,
	znnCliPillarUncollected,
	znnCliPillarCollect,
	znnCliPillarDelegate,
	znnCliPillarUndelegate,
	znnCliSporkList,
	znnCliSporkCreate,
	znnCliSporkActivate,
	znnCliSentinelUncollected,
	znnCliSentinelCollect,
	znnCliStakeList,
	znnCliStakeRegister,
	znnCliStakeRevoke,
	znnCliStakeUncollected,
	znnCliStakeCollect,
}

var znnCliCommand = cli.Command{
	Name:        "znn-cli",
	Usage:       "A port of znn_cli_dart",
	Subcommands: znnCliSubcommands,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "url",
			Aliases:     []string{"u"},
			Usage:       "Provide a websocket znnd connection URL with a port",
			Value:       "ws://127.0.0.1:35998",
			Destination: &url,
		},
		&cli.IntFlag{
			Name:        "chainId",
			Aliases:     []string{"n"},
			Usage:       "Specify the chain idendtifier to use",
			Value:       1,
			Destination: &chainId,
		},
		&cli.StringFlag{
			Name:    "passphrase",
			Aliases: []string{"p"},
			Usage:   "use this passphrase for the keyStore or enter it manually in a secure way",
		},
		&cli.StringFlag{
			Name:    "keyStore",
			Aliases: []string{"k"},
			Usage:   "Select the local keyStore",
		},
		&cli.IntFlag{
			Name:    "index",
			Aliases: []string{"i"},
			Usage:   "Address index",
			Value:   0,
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Prints detailed information about the action that it performs",
		},
	},
}
