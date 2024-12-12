package main

import (
	"fmt"

	"github.com/hypercore-one/go-zdk/utils"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/constants"
)

var znnCliSporkList = &cli.Command{
	Name:  "spork.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("spork.list")
			return nil
		}

		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		sporkList, err := z.Embedded.Spork.GetAll(0, rpcMaxPageSize)
		if err != nil {
			fmt.Println("Error getting spork list:", err)
			return err
		}
		if len(sporkList.List) == 0 {
			fmt.Println("No sporks found")
		} else {
			fmt.Println("Sporks:")
			for _, s := range sporkList.List {
				fmt.Printf("Name: %v\n", s.Name)
				fmt.Printf("  Description: %v\n", s.Description)
				fmt.Printf("  Activated: %v\n", s.Activated)
				if s.Activated {
					fmt.Printf("  EnforcementHeight: %v\n", s.EnforcementHeight)
				}
				fmt.Printf("  Hash: %v\n", s.Id)
			}
		}

		return nil
	},
}

var znnCliSporkCreate = &cli.Command{
	Name:  "spork.create",
	Usage: "name description",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 2 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("spork.list name description")
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

		name := cCtx.Args().Get(0)
		if len(name) < constants.SporkNameMinLength || len(name) > constants.SporkNameMaxLength {
			fmt.Println("Spork name must be", constants.SporkNameMinLength, "to", constants.SporkNameMaxLength, "characters in length")
			return nil
		}
		description := cCtx.Args().Get(1)
		if len(description) > constants.SporkDescriptionMaxLength {
			fmt.Println("Spork description cannot exceed", constants.SporkDescriptionMaxLength, "characters in length")
		}

		template, err := z.Embedded.Spork.Create(name, description)
		if err != nil {
			fmt.Println("Error templating spork create tx:", err)
			return err
		}
		fmt.Println("Creating spork...")
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending spork create tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliSporkActivate = &cli.Command{
	Name:  "spork.activate",
	Usage: "id",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 1 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("spork.activate id")
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

		id := types.HexToHashPanic(cCtx.Args().Get(0))

		template, err := z.Embedded.Spork.Activate(id)
		if err != nil {
			fmt.Println("Error templating spork activate tx:", err)
			return err
		}
		fmt.Println("Activating spork...")
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending spork activate tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}
