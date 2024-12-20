package main

import (
	"fmt"

	"github.com/hypercore-one/go-zdk/utils"
	"github.com/urfave/cli/v2"
)

var znnCliSentinelUncollected = &cli.Command{
	Name:  "sentinel.uncollected",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("sentinel.uncollected")
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
		uncollected, err := z.Embedded.Sentinel.GetUncollectedReward(kp.Address())
		if err != nil {
			fmt.Println("Error getting uncollected sentinel reward(s):", err)
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

var znnCliSentinelCollect = &cli.Command{
	Name:  "sentinel.collect",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("sentinel.collect")
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
		template, err := z.Embedded.Sentinel.CollectReward()
		if err != nil {
			fmt.Println("Error templating sentinel collect tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending sentinel collect tx:", err)
			return err
		}

		fmt.Println("Done")
		fmt.Println("Use 'receiveAll' to collect your Sentinel reward(s) after 1 momentum")
		return nil
	},
}
