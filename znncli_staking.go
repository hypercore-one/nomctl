package main

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/hypercore-one/go-zdk/utils"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/constants"
)

var znnCliStakeList = &cli.Command{
	Name:  "stake.list",
	Usage: "[pageIndex pageSize]",
	Action: func(cCtx *cli.Context) error {
		if !(cCtx.NArg() == 0 || cCtx.NArg() == 2) {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.collect [pageIndex pageSize]")
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

		if cCtx.NArg() == 2 {
			pageIndex, err = strconv.Atoi(cCtx.Args().Get(0))
			if err != nil {
				fmt.Println("Error:", err)
				return nil
			}
			pageSize, err = strconv.Atoi(cCtx.Args().Get(1))
			if err != nil {
				fmt.Println("Error:", err)
				return nil
			}
		}

		if !areValidPageVars(pageIndex, pageSize) {
			return nil
		}

		currentTime := time.Now().Unix()
		stakeList, err := z.Embedded.Stake.GetEntriesByAddress(kp.Address(), uint32(pageIndex), uint32(pageSize))
		if err != nil {
			fmt.Println("Error getting stake list:", err)
			return err
		}

		if stakeList.Count > 0 {
			fmt.Printf("Showing %v out of a total of %v staking entries\n", len(stakeList.Entries), stakeList.Count)
		} else {
			fmt.Println("No staking entries found")
		}

		for _, stakeEntry := range stakeList.Entries {
			fmt.Printf("Stake id %v with amount %v ZNN\n", stakeEntry.Id, formatAmount(stakeEntry.Amount, ZnnDecimals))
			if stakeEntry.ExpirationTimestamp > currentTime {
				// TODO format duration
				fmt.Printf("    Can be revoked in %v seconds\n", stakeEntry.ExpirationTimestamp-currentTime)
			} else {
				fmt.Println("     Can be revoked now")
			}
		}

		return nil
	},
}

var znnCliStakeRegister = &cli.Command{
	Name:  "stake.register",
	Usage: "amount duration (in months)",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 2 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.register amount duration (in months)")
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

		amount := big.NewInt(0)
		amount, ok := amount.SetString(cCtx.Args().Get(0), 10)
		if !ok {
			fmt.Println("Error: bad amount")
			return err
		}
		amount = amount.Mul(amount, big.NewInt(constants.Decimals))
		if amount.Cmp(constants.StakeMinAmount) == -1 {
			fmt.Printf("Invalid amount: %v ZNN. Minimum staking amount is %v\n", formatAmount(amount, ZnnDecimals), formatAmount(constants.StakeMinAmount, ZnnDecimals))
			return nil
		}

		duration, err := strconv.Atoi(cCtx.Args().Get(1))
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
		if duration < 1 || duration > 12 {
			fmt.Printf("Invalid duration: %v month. It must be between 1 and 12\n", duration)
		}

		info, err := z.Ledger.GetAccountInfoByAddress(kp.Address())
		if err != nil {
			fmt.Println("Error getting account info:", err)
			return err
		}

		if info.BalanceInfoMap[z.ZToken()].Balance.Cmp(constants.StakeMinAmount) == -1 {
			fmt.Println("Not enough ZNN to stake")

		}

		template, err := z.Embedded.Stake.Stake(int64(duration)*constants.StakeTimeUnitSec, amount)
		if err != nil {
			fmt.Println("Error templating stake register tx:", err)
			return err
		}
		fmt.Printf("Staking %v ZNN for %v month(s)\n", formatAmount(amount, ZnnDecimals), duration)
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending stake register tx:", err)
			return err
		}

		fmt.Println("Done")
		return nil
	},
}

var znnCliStakeRevoke = &cli.Command{
	Name:  "stake.revoke",
	Usage: "id",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 1 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.collect id")
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

		stakeId := types.HexToHashPanic(cCtx.Args().Get(0))

		pageIndex := 0
		pageSize := 25
		found := false
		gotError := false

		// TODO why is there no simple way to look up a stake entry

		stakeEntries, err := z.Embedded.Stake.GetEntriesByAddress(kp.Address(), uint32(pageIndex), uint32(pageSize))
		if err != nil {
			fmt.Println("Error getting stake list:", err)
			return err
		}
		for len(stakeEntries.Entries) > 0 {

			for _, s := range stakeEntries.Entries {
				if s.Id == stakeId {
					found = true
					m, err := z.Ledger.GetFrontierMomentum()
					if err != nil {
						fmt.Println("Error getting frontier momentum:", err)
						return err
					}

					if uint64(s.ExpirationTimestamp) > m.TimestampUnix {
						fmt.Println("Error! Stake entry can not be cancelled yet")
						gotError = true
					}
					break
				}
				if found {
					break
				}
			}

			pageIndex++
			stakeEntries, err = z.Embedded.Stake.GetEntriesByAddress(kp.Address(), uint32(pageIndex), uint32(pageSize))
			if err != nil {
				fmt.Println("Error getting stake list:", err)
				return err
			}
		}

		if !found {
			fmt.Println("Error! Stake entry was not found")
			return nil
		}

		if gotError {
			return nil
		}

		fmt.Printf("Canceling stake entry with id %v\n", stakeId)
		template, err := z.Embedded.Stake.Cancel(stakeId)
		if err != nil {
			fmt.Println("Error templating stake cancel tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending stake cancel tx:", err)
			return err
		}
		fmt.Println("Done")
		return nil
	},
}

var znnCliStakeUncollected = &cli.Command{
	Name:  "stake.uncollected",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.uncollected")
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
		uncollected, err := z.Embedded.Stake.GetUncollectedReward(kp.Address())
		if err != nil {
			fmt.Println("Error getting uncollected stake reward(s):", err)
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

var znnCliStakeCollect = &cli.Command{
	Name:  "stake.collect",
	Usage: "",
	Action: func(cCtx *cli.Context) error {
		if cCtx.NArg() != 0 {
			fmt.Println("Incorrect number of arguments. Expected:")
			fmt.Println("stake.collect")
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
		template, err := z.Embedded.Stake.CollectReward()
		if err != nil {
			fmt.Println("Error templating stake collect tx:", err)
			return err
		}
		_, err = utils.Send(z, template, kp, false)
		if err != nil {
			fmt.Println("Error sending stake collect tx:", err)
			return err
		}

		fmt.Println("Done")
		fmt.Println("Use 'receiveAll' to collect your stake reward(s) after 1 momentum")
		return nil
	},
}
