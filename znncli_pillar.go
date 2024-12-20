package main

import (
	"fmt"

	"github.com/hypercore-one/go-zdk/utils"
	"github.com/urfave/cli/v2"
)

var znnCliPillarList = &cli.Command{
	Name:  "pillar.list",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.list")
			return nil
		}

		z, err := connect(url, chainId)
		if err != nil {
			fmt.Println("Error connecting to Zenon Network:", err)
			return err
		}
		pillarInfoList, err := z.Embedded.Pillar.GetAll(0, rpcMaxPageSize)
		if err != nil {
			fmt.Println("Error getting pillar list:", err)
			return err
		}

		for _, p := range pillarInfoList.List {
			fmt.Printf("#%d Pillar %s has a delegated weight of %s ZNN\n", p.Rank+1, p.Name, formatAmount(p.Weight, ZnnDecimals))
			fmt.Printf("    Producer address %s\n", p.BlockProducingAddress)
			fmt.Printf("    Momentums %d / %d\n", p.CurrentStats.ProducedMomentums, p.CurrentStats.ExpectedMomentums)
		}
		return nil
	},
}

var znnCliPillarUncollected = &cli.Command{
	Name:  "pillar.uncollected",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.uncollected")
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
		uncollected, err := z.Embedded.Pillar.GetUncollectedReward(kp.Address())
		if err != nil {
			fmt.Println("Error getting uncollected pillar reward(s):", err)
			return err
		}
		if uncollected.Znn.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Znn, ZnnDecimals), "ZNN")
		}
		if uncollected.Qsr.Sign() != 0 {
			fmt.Println(formatAmount(uncollected.Qsr, ZnnDecimals), "QSR")
		}
		if uncollected.Znn.Sign() == 0 && uncollected.Qsr.Sign() == 0 {
			fmt.Println("No rewards to collect")
		}

		return nil
	},
}

var znnCliPillarCollect = &cli.Command{
	Name:  "pillar.collect",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.collect")
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
		template, err := z.Embedded.Pillar.CollectReward()
		if err != nil {
			fmt.Println("Error templating pillar collect tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending pillar collect tx:", err)
			return err
		}

		fmt.Println("Done")
		fmt.Println("Use 'receiveAll' to collect your Pillar reward(s) after 1 momentum")
		return nil
	},
}

var znnCliPillarDelegate = &cli.Command{
	Name:  "pillar.delegate",
	Usage: "name",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 1 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.delegate name")
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

		pillar := cCtx.Args().Get(0)

		template, err := z.Embedded.Pillar.Delegate(pillar)
		if err != nil {
			fmt.Println("Error templating pillar delegate tx:", err)
			return err
		}
		fmt.Println("Delegating to Pillar", pillar)
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending pillar delegate tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliPillarUndelegate = &cli.Command{
	Name:  "pillar.undelegate",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("pillar.undelegate")
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
		template, err := z.Embedded.Pillar.Undelegate()
		if err != nil {
			fmt.Println("Error templating pillar undelegate tx:", err)
			return err
		}
		fmt.Println("Undelegating ...")
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending pillar undelegate tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}
