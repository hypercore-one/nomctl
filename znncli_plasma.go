package main

import (
	"fmt"
	"math/big"

	"github.com/hypercore-one/go-zdk/utils"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/constants"
)

var znnCliPlasmaList = &cli.Command{
	Name:  "plasma.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("plasma.list")
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

		pageIndex := 0
		pageSize := 25

		fusions, err := z.Embedded.Plasma.GetEntriesByAddress(kp.Address(), uint32(pageIndex), uint32(pageSize))
		if err != nil {
			fmt.Println("Error getting plasma list:", err)
			return err
		}

		if fusions.Count > 0 {
			fmt.Printf("Fusing %s QSR for Plasma in %v entries\n", formatAmount(fusions.QsrAmount, QsrDecimals), fusions.Count)
			for _, f := range fusions.Fusions {
				fmt.Printf("  %v QSR for %v\n", formatAmount(f.QsrAmount, QsrDecimals), f.Beneficiary)
				fmt.Printf("Can be cancelled at momentum height: %v. Use id %v to cancel\n", f.ExpirationHeight, f.Id)
			}
		} else {
			fmt.Println("No Plasma fusion entries found")
		}

		return nil
	},
}

var znnCliPlasmaGet = &cli.Command{
	Name:  "plasma.get",
	Usage: "[address]",
	Action: func(cCtx *cli.Context) error {
		if !(cCtx.NArg() == 0 || cCtx.NArg() == 1) {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("plasma.get [address]")
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

		address := kp.Address()
		if cCtx.NArg() == 1 {
			address = types.ParseAddressPanic(cCtx.Args().Get(0))
		}

		plasmaInfo, err := z.Embedded.Plasma.Get(address)
		if err != nil {
			fmt.Println("Error getting plasma info:", err)
			return err
		}
		currentPlasma := plasmaInfo.CurrentPlasma
		maxPlasma := plasmaInfo.MaxPlasma
		formattedQsrAmount := formatAmount(plasmaInfo.QsrAmount, QsrDecimals)

		fmt.Printf("%s has %v/%v plasma with %v QSR fused.\n", address, currentPlasma, maxPlasma, formattedQsrAmount)
		return nil
	},
}

var znnCliPlasmaFuse = &cli.Command{
	Name:  "plasma.fuse",
	Usage: "toAddress amount",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 2 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("plasma.fuse toAddress amount")
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

		toAddress := types.ParseAddressPanic(cCtx.Args().Get(0))
		amount := big.NewInt(0)
		amount, ok := amount.SetString(cCtx.Args().Get(1), 10)
		if !ok {
			fmt.Println("Error: bad amount")
			return nil
		}
		amount = amount.Mul(amount, big.NewInt(constants.Decimals))

		if amount.Cmp(constants.FuseMinAmount) == -1 {
			fmt.Printf("Invalid amount: %v QSR. Minimum fusing amount is %v\n", formatAmount(amount, QsrDecimals), formatAmount(constants.FuseMinAmount, QsrDecimals))
			return nil
		}

		rem := big.NewInt(0)
		rem = rem.Rem(amount, constants.TokenIssueAmount)
		if rem.Cmp(big.NewInt(0)) != 0 {
			fmt.Printf("Error! Amount has to be integer")
			return nil
		}

		fmt.Printf("Fusing %v QSR to %v\n", formatAmount(amount, QsrDecimals), toAddress)
		template, err := z.Embedded.Plasma.Fuse(toAddress, amount)
		if err != nil {
			fmt.Println("Error creating fusing plasma template:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error fusing plasma:", err)
			return err
		}
		fmt.Println("Done")
		return nil
	},
}

var znnCliPlasmaCancel = &cli.Command{
	Name:  "plasma.cancel",
	Usage: "id",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 1 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("plasma.cancel id")
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

		fuseId := types.HexToHashPanic(cCtx.Args().Get(0))

		pageIndex := 0
		pageSize := 25
		found := false
		gotError := false

		// TODO why is there no simple way to look up a fusion entry

		fusions, err := z.Embedded.Plasma.GetEntriesByAddress(kp.Address(), uint32(pageIndex), uint32(pageSize))
		if err != nil {
			fmt.Println("Error getting plasma list:", err)
			return err
		}
		for len(fusions.Fusions) > 0 {

			for _, f := range fusions.Fusions {
				if f.Id == fuseId {
					found = true
					m, err := z.Ledger.GetFrontierMomentum()
					if err != nil {
						fmt.Println("Error getting frontier momentum:", err)
						return err
					}

					if f.ExpirationHeight > m.Height {
						fmt.Println("Error! Fuse entry can not be cancelled yet")
						gotError = true
					}
					break
				}
				if found {
					break
				}
			}

			pageIndex++
			fusions, err = z.Embedded.Plasma.GetEntriesByAddress(kp.Address(), uint32(pageIndex), uint32(pageSize))
			if err != nil {
				fmt.Println("Error getting plasma list:", err)
				return err
			}
		}

		if !found {
			fmt.Println("Error! Fuse entry was not found")
			return nil
		}

		if gotError {
			return nil
		}

		fmt.Printf("Canceling Plasma fuse entry with id %v\n", fuseId)
		template, err := z.Embedded.Plasma.Cancel(fuseId)
		if err != nil {
			fmt.Println("Error templating plasma cancel tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending plasma cancel tx:", err)
			return err
		}
		fmt.Println("Done")
		return nil
	},
}
