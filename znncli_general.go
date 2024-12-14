package main

import (
	"fmt"
	"math/big"

	"github.com/hypercore-one/go-zdk/utils"
	"github.com/hypercore-one/go-zdk/utils/template"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
)

// TODO message data
var znnCliSend = &cli.Command{
	Name:  "send",
	Usage: "toAddress amount zts",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 3 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("send toAddress amount zts")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		toAddress, err := types.ParseAddress(cCtx.Args().Get(0))
		if err != nil {
			fmt.Println("Error bad toAddress:", err)
			return err
		}

		zts, err := getTokenStandard(cCtx.Args().Get(2))
		if err != nil {
			fmt.Println("Error bad zts:", err)
			return err
		}

		token, err := z.Embedded.Token.GetByZts(zts)
		if err != nil {
			fmt.Println("Error fetching zts:", err)
			return err
		}

		// TODO handle decimal amounts
		amount := big.NewInt(0)
		amount, ok := amount.SetString(cCtx.Args().Get(1), 10)
		if !ok {
			fmt.Println("Error bad amount")
			return nil
		}

		amount = amount.Mul(amount, big.NewInt(10^int64(token.Decimals)))

		tmpl := template.Send(z.ProtocolVersion(), z.ChainIdentifier(), toAddress, zts, amount, []byte{})
		_, err = utils.Send(z, tmpl, kp, false)
		if err != nil {
			fmt.Println("Error sending tx", err)
			return err
		}

		return nil
	},
}

var znnCliUnreceived = &cli.Command{
	Name:  "unreceived",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("unreceived")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		unreceived, err := z.Ledger.GetUnreceivedBlocksByAddress(kp.Address(), 0, 5)
		if err != nil {
			fmt.Println("Error fetching unreceived txs:", err)
			return err
		}
		if len(unreceived.List) == 0 {
			fmt.Println("Nothing to receive")
			return nil
		} else {
			if unreceived.More {
				fmt.Println("You have more than", unreceived.Count, "transaction(s) to receive")
			} else {
				fmt.Println("You have", unreceived.Count, "transaction(s) to receive")
			}
		}
		fmt.Println("Showing the first", unreceived.Count)
		for _, block := range unreceived.List {
			fmt.Println("Unreceived", formatAmount(block.Amount, block.TokenInfo.Decimals), block.TokenInfo.TokenSymbol, "from", block.Address, "Use the hash", block.Hash, "to receive")
		}
		return nil
	},
}

var znnCliReceiveAll = &cli.Command{
	Name:  "receiveAll",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("receiveAll")
			return nil
		}

		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			fmt.Println("Error getting signer:", err)
			return err
		}
		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}

		unreceived, err := z.Ledger.GetUnreceivedBlocksByAddress(kp.Address(), 0, 5)
		if err != nil {
			fmt.Println("Error fetching unreceived txs:", err)
			return err
		}
		if len(unreceived.List) == 0 {
			fmt.Println("Nothing to receive")
			return nil
		} else {
			if unreceived.More {
				fmt.Println("You have more than", unreceived.Count, "transaction(s) to receive")
			} else {
				fmt.Println("You have", unreceived.Count, "transaction(s) to receive")
			}
		}
		fmt.Println("Please wait ...")

		for unreceived.Count > 0 {
			for _, block := range unreceived.List {
				temp := template.Receive(z.ProtocolVersion(), z.ChainIdentifier(), block.Hash)
				_, err = utils.Send(z, temp, kp, false)
				if err != nil {
					fmt.Println("Error receiving txs:", err)
					return err
				}
			}
			unreceived, err = z.Ledger.GetUnreceivedBlocksByAddress(kp.Address(), 0, 5)
			if err != nil {
				fmt.Println("Error fetching unreceived txs:", err)
				return err
			}
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliBalance = &cli.Command{
	Name: "balance",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("balance")
			return nil
		}
		kp, err := getZnnCliSigner(walletDir, cCtx)
		if err != nil {
			return err
		}
		if kp == nil {
			return nil
		}

		z, err := connect(url, chainId)
		if err != nil {
			return err
		}
		info, err := z.Ledger.GetAccountInfoByAddress(kp.Address())
		if err != nil {
			return err
		}
		fmt.Println("Balance for account-chain", kp.Address().String(), "having height", info.AccountHeight)
		if len(info.BalanceInfoMap) == 0 {
			fmt.Println("  No coins or tokens at address", kp.Address().String())
		}
		for zts, entry := range info.BalanceInfoMap {
			fmt.Println(" ", formatAmount(entry.Balance, entry.TokenInfo.Decimals), entry.TokenInfo.TokenSymbol, entry.TokenInfo.TokenDomain, zts.String())
		}
		return nil
	},
}

var znnCliFrontierMomentum = &cli.Command{
	Name: "frontierMomentum",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("frontierMomentum")
			return nil
		}
		z, err := connect(url, chainId)
		if err != nil {
			return err
		}
		m, err := z.Ledger.GetFrontierMomentum()
		if err != nil {
			return err
		}
		fmt.Println("Momentum height:", m.Height)
		fmt.Println("Momentum hash:", m.Hash.String())
		fmt.Println("Momentum previousHash:", m.PreviousHash.String())
		fmt.Println("Momentum timestamp:", m.TimestampUnix)
		return nil
	},
}
