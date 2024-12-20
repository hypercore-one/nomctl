package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/chain/genesis"
	"github.com/zenon-network/go-zenon/common/crypto"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"
)

type QubePadRegistration struct {
	Name     string
	Owner    types.Address
	Withdraw types.Address
	Producer types.Address
}

var (
	QubepadFlag = cli.StringFlag{
		Name:     "qubepad-csv",
		Usage:    "Path to qubepad csv export",
		Required: true,
	}

	generateHyperQubeGenesisCommand = cli.Command{
		Action:    hyperqubeGenesisAction,
		Name:      "generate-hyperqube-genesis",
		Usage:     "Generates config for hyperqube",
		ArgsUsage: " ",

		Flags: []cli.Flag{
			&DataPathFlag,
			&GenesisFileFlag,
			&QubepadFlag,
		},
	}
)

func hyperqubeGenesisAction(ctx *cli.Context) error {

	file, err := os.Open(ctx.String(QubepadFlag.Name))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	_, _ = reader.Read() // skip headers
	data, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	pillars := make([]QubePadRegistration, 0)
	for _, row := range data {
		if row[4] != "" {
			fmt.Printf("Forming pillar: %v\n", row[4])
			pillars = append(pillars, QubePadRegistration{
				Name:     row[4],
				Owner:    types.ParseAddressPanic(row[5]),
				Withdraw: types.ParseAddressPanic(row[6]),
				Producer: types.ParseAddressPanic(row[7]),
			})
		}
	}

	if err := createHQZGenesis(ctx, pillars); err != nil {
		return err
	}

	return nil
}

func createHQZGenesis(ctx *cli.Context, pillars []QubePadRegistration) error {

	// STATIC CONFIG

	utilZ := definition.TokenInfo{
		Decimals:      8,
		IsBurnable:    true,
		IsMintable:    true,
		IsUtility:     true,
		MaxSupply:     big.NewInt(9007199254740991),
		Owner:         types.TokenContract,
		TokenDomain:   "hyperqube.network",
		TokenName:     "utilZ",
		TokenStandard: utilZ,
		TokenSymbol:   "utilZ",
		TotalSupply:   big.NewInt(0),
	}
	utilQ := definition.TokenInfo{
		Decimals:      8,
		IsBurnable:    true,
		IsMintable:    true,
		IsUtility:     true,
		MaxSupply:     big.NewInt(9007199254740991),
		Owner:         types.TokenContract,
		TokenDomain:   "hyperqube.network",
		TokenName:     "utilQ",
		TokenStandard: utilQ,
		TokenSymbol:   "utilQ",
		TotalSupply:   big.NewInt(0),
	}

	// activate sporks for accelerator-z, htlc, and deactivating pillar registration
	// create spork for bridge but do not activate

	genesisSporksMap := make(map[types.Hash]bool)
	for sporkId, status := range types.ImplementedSporksMap {
		genesisSporksMap[sporkId] = status
	}

	genesisSporks := make([]*definition.Spork, 0)
	genesisSporks = append(genesisSporks, &definition.Spork{
		Id:                types.AcceleratorSpork.SporkId,
		Name:              "az",
		Description:       "az",
		Activated:         true,
		EnforcementHeight: 0,
	})
	genesisSporks = append(genesisSporks, &definition.Spork{
		Id:                types.HtlcSpork.SporkId,
		Name:              "htlc",
		Description:       "htlc",
		Activated:         true,
		EnforcementHeight: 0,
	})
	genesisSporks = append(genesisSporks, &definition.Spork{
		Id:          types.BridgeAndLiquiditySpork.SporkId,
		Name:        "bridge-liq",
		Description: "bridge-liq",
		Activated:   false,
		// confirm enforcement hieght does nothing when not activated
		EnforcementHeight: 0,
	})

	//sha3.256(hyperqube_z spork deactivate pillar registration)
	genesisSporks = append(genesisSporks, &definition.Spork{
		Id:                types.HexToHashPanic("c35c80695e6f1739ce19bd9b31e4a6702335fafd643139eb73b76541be2ca9e4"),
		Name:              "hyperqube-no-pillar-reg",
		Description:       "hyperqube-no-pillar-reg",
		Activated:         true,
		EnforcementHeight: 0,
	})

	sporkAddress, _ := types.ParseAddress("z1qpg8v63m534t2vv09yzndv9gu9t6gyrvq3n6qv")

	gen := genesis.GenesisConfig{
		ChainIdentifier:     26,
		ExtraData:           "HYPERQUBE Z UNIFORM 60",
		GenesisTimestampSec: time.Now().Unix(),
		SporkAddress:        &sporkAddress,

		PillarConfig: &genesis.PillarContractConfig{
			Delegations:   []*definition.DelegationInfo{},
			LegacyEntries: []*definition.LegacyPillarEntry{},
			Pillars:       []*definition.PillarInfo{},
		},
		TokenConfig: &genesis.TokenContractConfig{
			Tokens: []*definition.TokenInfo{
				&utilZ,
				&utilQ,
			}},
		PlasmaConfig: &genesis.PlasmaContractConfig{
			Fusions: []*definition.FusionInfo{}},
		SwapConfig: &genesis.SwapContractConfig{
			Entries: []*definition.SwapAssets{}},
		SporkConfig: &genesis.SporkConfig{
			Sporks: genesisSporks,
		},
		GenesisBlocks: &genesis.GenesisBlocksConfig{
			Blocks: []*genesis.GenesisBlockConfig{},
		}}

	gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &genesis.GenesisBlockConfig{
		Address: types.AcceleratorContract,
		BalanceList: map[types.ZenonTokenStandard]*big.Int{
			utilZ.TokenStandard: big.NewInt(1000000 * constants.Decimals),
			utilQ.TokenStandard: big.NewInt(10000000 * constants.Decimals),
		},
	})
	utilZ.TotalSupply.Add(utilZ.TotalSupply, big.NewInt(1000000*constants.Decimals))
	utilQ.TotalSupply.Add(utilQ.TotalSupply, big.NewInt(10000000*constants.Decimals))

	// DYNAMIC CONFIG

	totalPillarZ := big.NewInt(0)
	totalPlasmaQ := big.NewInt(0)

	for _, p := range pillars {
		gen.PillarConfig.Pillars = append(gen.PillarConfig.Pillars, &definition.PillarInfo{
			Name:                         p.Name,
			Amount:                       big.NewInt(15000 * constants.Decimals),
			BlockProducingAddress:        p.Producer,
			StakeAddress:                 p.Owner,
			RewardWithdrawAddress:        p.Withdraw,
			PillarType:                   1,
			RevokeTime:                   0,
			GiveBlockRewardPercentage:    0,
			GiveDelegateRewardPercentage: 100,
		})
		utilZ.TotalSupply.Add(utilZ.TotalSupply, big.NewInt(15000*constants.Decimals))
		totalPillarZ.Add(totalPillarZ, big.NewInt(15000*constants.Decimals))

		gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &genesis.GenesisBlockConfig{
			Address: p.Owner,
			BalanceList: map[types.ZenonTokenStandard]*big.Int{
				utilZ.TokenStandard: big.NewInt(10000 * constants.Decimals),
				utilQ.TokenStandard: big.NewInt(100000 * constants.Decimals),
			},
		})
		utilZ.TotalSupply.Add(utilZ.TotalSupply, big.NewInt(10000*constants.Decimals))
		utilQ.TotalSupply.Add(utilQ.TotalSupply, big.NewInt(100000*constants.Decimals))

		fuseAmount := big.NewInt(1000 * constants.Decimals)

		fusionO := definition.FusionInfo{
			Owner:            p.Owner,
			Id:               types.Hash(crypto.Hash(append(p.Owner.Bytes(), byte('H')))),
			Amount:           fuseAmount,
			ExpirationHeight: 1,
			Beneficiary:      p.Owner,
		}
		gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusionO)
		utilQ.TotalSupply.Add(utilQ.TotalSupply, fuseAmount)
		totalPlasmaQ.Add(totalPlasmaQ, fuseAmount)

		fusionW := definition.FusionInfo{
			Owner:            p.Owner,
			Id:               types.Hash(crypto.Hash(append(p.Withdraw.Bytes(), byte('Q')))),
			Amount:           fuseAmount,
			ExpirationHeight: 1,
			Beneficiary:      p.Withdraw,
		}
		gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusionW)
		utilQ.TotalSupply.Add(utilQ.TotalSupply, fuseAmount)
		totalPlasmaQ.Add(totalPlasmaQ, fuseAmount)

		fusionP := definition.FusionInfo{
			Owner:            p.Owner,
			Id:               types.Hash(crypto.Hash(append(p.Producer.Bytes(), byte('Z')))),
			Amount:           fuseAmount,
			ExpirationHeight: 1,
			Beneficiary:      p.Producer,
		}
		gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusionP)
		utilQ.TotalSupply.Add(utilQ.TotalSupply, fuseAmount)
		totalPlasmaQ.Add(totalPlasmaQ, fuseAmount)
	}

	gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &genesis.GenesisBlockConfig{
		Address: types.PillarContract,
		BalanceList: map[types.ZenonTokenStandard]*big.Int{
			utilZ.TokenStandard: totalPillarZ,
		},
	})

	gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &genesis.GenesisBlockConfig{
		Address: types.PlasmaContract,
		BalanceList: map[types.ZenonTokenStandard]*big.Int{
			utilQ.TokenStandard: totalPlasmaQ,
		},
	})

	file, _ := json.MarshalIndent(gen, "", " ")

	path := "genesis.json"
	if ctx.IsSet(GenesisFileFlag.Name) {
		path = ctx.String(GenesisFileFlag.Name)
	}
	_ = os.WriteFile(path, file, 0644)

	return nil
}
