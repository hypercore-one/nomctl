package main

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/tyler-smith/go-bip39"
	"github.com/urfave/cli/v2"
	"github.com/zenon-network/go-zenon/chain/genesis"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/node"
	"github.com/zenon-network/go-zenon/p2p"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"
	"github.com/zenon-network/go-zenon/wallet"
)

var (
	DataPathFlag = cli.StringFlag{
		Name:  "data",
		Usage: "Path to the main zenon data folder. Used for store all files.",
		Value: node.DefaultDataDir(),
	}
	WalletDirFlag = cli.StringFlag{
		Name:  "wallet",
		Usage: "Directory for the wallet.",
		Value: "DataPath/wallet",
	}
	GenesisFileFlag = cli.StringFlag{
		Name:  "genesis",
		Usage: "Path to genesis file. Used to override embedded genesis from the binary source-code.",
		Value: "DataPath/genesis.json",
	}

	GenesisBlockFlag = cli.StringSliceFlag{
		Name:  "genesis-block",
		Usage: "<address>/<ZnnAmount>/<QsrAmount>",
	}

	GenesisFusionFlag = cli.StringSliceFlag{
		Name:  "genesis-fusion",
		Usage: "<address>/<QsrAmount>",
	}

	// TODO
	SporkAddressFlag = cli.StringFlag{
		Name:  "spork-address",
		Usage: "<address>",
	}

	// TODO
	GenesisSporkFlag = cli.StringSliceFlag{
		Name:  "genesis-spork",
		Usage: "<hashId>,<activationStatus: true,false>",
	}

	GenesisEZFlag = cli.BoolFlag{
		Name: "ez",
	}

	devnetCommand = cli.Command{
		Action:    devnetAction,
		Name:      "generate-devnet",
		Usage:     "Generates config for devnet",
		ArgsUsage: " ",
		Category:  "DEVELOPER COMMANDS",

		Flags: []cli.Flag{
			&DataPathFlag,
			&WalletDirFlag,
			&GenesisFileFlag,
			&GenesisBlockFlag,
			&GenesisFusionFlag,
			&SporkAddressFlag,
			&GenesisSporkFlag,
			&GenesisEZFlag,
		},
	}
)

func devnetAction(ctx *cli.Context) error {

	cfg := node.DefaultNodeConfig

	// 1: Apply flags, Overwrite the configuration file configuration

	if dataDir := ctx.String(DataPathFlag.Name); ctx.IsSet(DataPathFlag.Name) && len(dataDir) > 0 {
		cfg.DataPath = dataDir
	}

	// Wallet
	if walletDir := ctx.String(WalletDirFlag.Name); ctx.IsSet(WalletDirFlag.Name) && len(walletDir) > 0 {
		cfg.WalletPath = walletDir
	}

	if genesisFile := ctx.String(GenesisFileFlag.Name); ctx.IsSet(GenesisFileFlag.Name) && len(genesisFile) > 0 {
		cfg.GenesisFile = genesisFile
	}
	// validate custom flags
	if err := validateDevnetFlags(ctx); err != nil {
		return err
	}

	// 2: Make dir paths absolute
	if err := cfg.MakePathsAbsolute(); err != nil {
		return err
	}

	// 3: Check/Create dirs
	if err := checkCreatePaths(&cfg); err != nil {
		return err
	}

	// 4: Generate Producer,
	if err := createDevProducer(&cfg); err != nil {
		return err
	}

	// 5. Generate NetConfig
	// TODO add flag for IP address to generate seeders to share with others
	if err := createDevNet(&cfg); err != nil {
		return err
	}

	// 6. Generate Genesis Config
	if hyperqube {
		if err := createHQZDevGenesis(ctx, &cfg); err != nil {
			return err
		}
	} else {
		if err := createDevGenesis(ctx, &cfg); err != nil {
			return err
		}
	}

	// write config
	configPath := filepath.Join(cfg.DataPath, "config.json")
	file, _ := json.MarshalIndent(cfg, "", " ")
	_ = os.WriteFile(configPath, file, 0700)

	return nil
}

func checkCreatePaths(cfg *node.Config) error {
	// Abort if datapath already exists
	if _, err := os.Stat(cfg.DataPath); err == nil {
		return errors.New("datapath already exists")
	}
	if err := os.MkdirAll(cfg.DataPath, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.WalletPath, 0700); err != nil {
		return err
	}
	return nil
}

func createDevProducer(cfg *node.Config) error {
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)

	ks := &wallet.KeyStore{
		Entropy:  entropy,
		Seed:     bip39.NewSeed(mnemonic, ""),
		Mnemonic: mnemonic,
	}
	_, kp, _ := ks.DeriveForIndexPath(0)
	ks.BaseAddress = kp.Address

	// TODO make this random
	password := "Don'tTrust.Verify"
	kf, _ := ks.Encrypt(password)
	kf.Path = filepath.Join(cfg.WalletPath, ks.BaseAddress.String())
	kf.Write()

	producer := node.ProducerConfig{
		Address:     kp.Address.String(),
		Index:       0,
		KeyFilePath: kf.Path,
		Password:    password,
	}

	cfg.Producer = &producer

	return nil
}

func createDevNet(cfg *node.Config) error {
	// generate network key
	// ask for ip address via flag??

	privateKeyFile := filepath.Join(cfg.DataPath, p2p.DefaultNetPrivateKeyFile)

	key, err := crypto.GenerateKey()
	if err != nil {
		log.Crit("Failed to generate node key", "reason", err)
	}

	if err := crypto.SaveECDSA(privateKeyFile, key); err != nil {
		log.Error("Failed to persist node key", "reason", err)
	}

	cfg.Net.MinPeers = 0
	cfg.Net.MinConnectedPeers = 0
	cfg.Net.Seeders = []string{}
	return nil
}

func validateDevnetFlags(ctx *cli.Context) error {
	if ctx.IsSet(GenesisBlockFlag.Name) {
		input := ctx.StringSlice(GenesisBlockFlag.Name)
		exists := make(map[types.Address]bool)
		for _, s := range input {

			ss := strings.Split(s, "/")
			if len(ss) != 3 {
				return errors.New("genesis-block flags must be in the format --genesis-block=<address>/<znnAmount>/<qsrAmount>")
			}

			a, err := types.ParseAddress(ss[0])
			if err != nil {
				return err
			}
			if types.IsEmbeddedAddress(a) {
				return errors.New("genesis-block flag can only be set for user addresses")
			}

			z, err := strconv.ParseUint(ss[1], 10, 64)
			if err != nil {
				return err
			}
			q, err := strconv.ParseUint(ss[2], 10, 64)
			if err != nil {
				return err
			}

			if z == 0 && q == 0 {
				return errors.New("genesis-block znn and qsr amount cannot both be 0")
			}
			// TODO maximum? and check for total token supply exceeds cap

			if _, ok := exists[a]; ok {
				return errors.New("genesis-block addresses must be unique")
			}
			exists[a] = true
		}
	}

	if ctx.IsSet(GenesisFusionFlag.Name) {
		input := ctx.StringSlice(GenesisFusionFlag.Name)
		exists := make(map[types.Address]bool)
		for _, s := range input {

			ss := strings.Split(s, "/")
			if len(ss) != 2 {
				return errors.New("genesis-fusion flags must be in the format --genesis-fusion=<address>/<qsrAmount>")
			}

			a, err := types.ParseAddress(ss[0])
			if err != nil {
				return err
			}
			if types.IsEmbeddedAddress(a) {
				return errors.New("genesis-fusion flag can only be set for user addresses")
			}

			q, err := strconv.ParseUint(ss[1], 10, 64)
			if err != nil {
				return err
			}

			if q == 0 || q > 5000 {
				return errors.New("genesis-fusion amount must be between min:1 max:5000")
			}

			if _, ok := exists[a]; ok {
				return errors.New("genesis-fusion addresses must be unique")
			}
			exists[a] = true
		}
	}
	return nil
}

func createDevGenesis(ctx *cli.Context, cfg *node.Config) error {
	if cfg.GenesisFile == "" {
		cfg.GenesisFile = filepath.Join(cfg.DataPath, "genesis.json")
	}

	localPillar, _ := types.ParseAddress(cfg.Producer.Address)

	znnStandard := definition.TokenInfo{
		Decimals:      8,
		IsBurnable:    true,
		IsMintable:    true,
		IsUtility:     true,
		MaxSupply:     big.NewInt(9007199254740991),
		Owner:         types.TokenContract,
		TokenDomain:   "biginches.club",
		TokenName:     "tZNN",
		TokenStandard: types.ZnnTokenStandard,
		TokenSymbol:   "tZNN",
		TotalSupply:   big.NewInt(78713599988800),
	}
	qsrStandard := definition.TokenInfo{
		Decimals:      8,
		IsBurnable:    true,
		IsMintable:    true,
		IsUtility:     true,
		MaxSupply:     big.NewInt(9007199254740991),
		Owner:         types.TokenContract,
		TokenDomain:   "biginches.club",
		TokenName:     "tQSR",
		TokenStandard: types.QsrTokenStandard,
		TokenSymbol:   "tQSR",
		TotalSupply:   big.NewInt(772135999888000),
	}

	// by default activate all implemented sporks at height 0
	// can be overriden by --genesis-spork
	genesisSporksMap := make(map[types.Hash]bool)
	for sporkId, status := range types.ImplementedSporksMap {
		genesisSporksMap[sporkId] = status
	}
	// apply genesis sporks flag
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
		Id:                types.BridgeAndLiquiditySpork.SporkId,
		Name:              "bridge-liq",
		Description:       "bridge-liq",
		Activated:         true,
		EnforcementHeight: 0,
	})

	gen := genesis.GenesisConfig{
		ChainIdentifier:     321,
		ExtraData:           "/thank_you_bich_dao",
		GenesisTimestampSec: time.Now().Unix(),
		SporkAddress:        &localPillar,

		PillarConfig: &genesis.PillarContractConfig{
			Delegations:   []*definition.DelegationInfo{},
			LegacyEntries: []*definition.LegacyPillarEntry{},
			Pillars: []*definition.PillarInfo{
				{
					Name:                         "Local",
					Amount:                       big.NewInt(1500000000000),
					BlockProducingAddress:        localPillar,
					StakeAddress:                 localPillar,
					RewardWithdrawAddress:        localPillar,
					PillarType:                   1,
					RevokeTime:                   0,
					GiveBlockRewardPercentage:    0,
					GiveDelegateRewardPercentage: 100,
				},
			}},
		TokenConfig: &genesis.TokenContractConfig{
			Tokens: []*definition.TokenInfo{
				&znnStandard,
				&qsrStandard,
			}},
		PlasmaConfig: &genesis.PlasmaContractConfig{
			Fusions: []*definition.FusionInfo{}},
		SwapConfig: &genesis.SwapContractConfig{
			Entries: []*definition.SwapAssets{}},
		SporkConfig: &genesis.SporkConfig{
			Sporks: genesisSporks,
		},
		GenesisBlocks: &genesis.GenesisBlocksConfig{
			Blocks: []*genesis.GenesisBlockConfig{
				{
					Address: types.PillarContract,
					BalanceList: map[types.ZenonTokenStandard]*big.Int{
						types.ZnnTokenStandard: big.NewInt(1500000000000),
					},
				},
				{
					Address: types.AcceleratorContract,
					BalanceList: map[types.ZenonTokenStandard]*big.Int{
						types.ZnnTokenStandard: big.NewInt(77213599988800),
						types.QsrTokenStandard: big.NewInt(772135999888000),
					},
				},
			},
		}}

	// Genesis Blocks

	if ctx.Bool(GenesisEZFlag.Name) {
		znn := big.NewInt(100000 * constants.Decimals)
		qsr := big.NewInt(500000 * constants.Decimals)

		znnStandard.TotalSupply.Add(znnStandard.TotalSupply, znn)
		qsrStandard.TotalSupply.Add(qsrStandard.TotalSupply, qsr)
		block := genesis.GenesisBlockConfig{
			Address: localPillar,
			BalanceList: map[types.ZenonTokenStandard]*big.Int{
				types.ZnnTokenStandard: znn,
				types.QsrTokenStandard: qsr,
			},
		}
		gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &block)
	}

	if ctx.IsSet(GenesisBlockFlag.Name) {
		input := ctx.StringSlice(GenesisBlockFlag.Name)
		for _, s := range input {

			ss := strings.Split(s, "/")
			a, _ := types.ParseAddress(ss[0])
			z, _ := strconv.ParseInt(ss[1], 10, 64)
			q, _ := strconv.ParseInt(ss[2], 10, 64)
			znn := big.NewInt(z * constants.Decimals)
			qsr := big.NewInt(q * constants.Decimals)

			znnStandard.TotalSupply.Add(znnStandard.TotalSupply, znn)
			qsrStandard.TotalSupply.Add(qsrStandard.TotalSupply, qsr)
			block := genesis.GenesisBlockConfig{
				Address: a,
				BalanceList: map[types.ZenonTokenStandard]*big.Int{
					types.ZnnTokenStandard: znn,
					types.QsrTokenStandard: qsr,
				},
			}
			gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &block)

		}
	}

	if ctx.IsSet(GenesisFusionFlag.Name) || ctx.Bool(GenesisEZFlag.Name) {
		plasmaAddress := big.NewInt(0)

		qsr := big.NewInt(1000 * constants.Decimals)
		fusion := definition.FusionInfo{
			Owner:            localPillar,
			Id:               types.NewHash(localPillar.Bytes()),
			Amount:           qsr,
			ExpirationHeight: 1,
			Beneficiary:      localPillar,
		}
		gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusion)
		plasmaAddress.Add(plasmaAddress, qsr)
		qsrStandard.TotalSupply.Add(qsrStandard.TotalSupply, qsr)

		input := ctx.StringSlice(GenesisFusionFlag.Name)
		for _, s := range input {

			ss := strings.Split(s, "/")
			a, _ := types.ParseAddress(ss[0])
			q, _ := strconv.ParseInt(ss[1], 10, 64)
			qsr := big.NewInt(q * constants.Decimals)

			qsrStandard.TotalSupply.Add(qsrStandard.TotalSupply, qsr)
			plasmaAddress.Add(plasmaAddress, qsr)
			fusion := definition.FusionInfo{
				Owner:            a,
				Id:               types.NewHash(a.Bytes()),
				Amount:           qsr,
				ExpirationHeight: 1,
				Beneficiary:      a,
			}
			gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusion)
		}
		block := genesis.GenesisBlockConfig{
			Address: types.PlasmaContract,
			BalanceList: map[types.ZenonTokenStandard]*big.Int{
				types.QsrTokenStandard: plasmaAddress,
			},
		}
		gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &block)

	}

	file, _ := json.MarshalIndent(gen, "", " ")
	_ = os.WriteFile(cfg.GenesisFile, file, 0644)

	return nil
}

func createHQZDevGenesis(ctx *cli.Context, cfg *node.Config) error {
	if cfg.GenesisFile == "" {
		cfg.GenesisFile = filepath.Join(cfg.DataPath, "genesis.json")
	}

	localPillar, _ := types.ParseAddress(cfg.Producer.Address)

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
		TotalSupply:   big.NewInt(78713599988800),
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
		TotalSupply:   big.NewInt(772135999888000),
	}

	// activate sporks for accelerator-z, htlc, and deactivating pillar registration
	// create spork for bridge but do not activate
	// can be overriden by --genesis-spork

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
		Id:          types.HexToHashPanic("c35c80695e6f1739ce19bd9b31e4a6702335fafd643139eb73b76541be2ca9e4"),
		Name:        "hyperqube-no-pillar-reg",
		Description: "hyperqube-no-pillar-reg",
		Activated:   false,
		// confirm enforcement hieght does nothing when not activated
		EnforcementHeight: 0,
	})

	gen := genesis.GenesisConfig{
		ChainIdentifier:     321,
		ExtraData:           "HYPERQUBE LOCAL UNIFORM 60",
		GenesisTimestampSec: time.Now().Unix(),
		SporkAddress:        &localPillar,

		PillarConfig: &genesis.PillarContractConfig{
			Delegations:   []*definition.DelegationInfo{},
			LegacyEntries: []*definition.LegacyPillarEntry{},
			Pillars: []*definition.PillarInfo{
				{
					Name:                         "Local",
					Amount:                       big.NewInt(1500000000000),
					BlockProducingAddress:        localPillar,
					StakeAddress:                 localPillar,
					RewardWithdrawAddress:        localPillar,
					PillarType:                   1,
					RevokeTime:                   0,
					GiveBlockRewardPercentage:    0,
					GiveDelegateRewardPercentage: 100,
				},
			}},
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
			Blocks: []*genesis.GenesisBlockConfig{
				{
					Address: types.PillarContract,
					BalanceList: map[types.ZenonTokenStandard]*big.Int{
						utilZ.TokenStandard: big.NewInt(1500000000000),
					},
				},
				{
					Address: types.AcceleratorContract,
					BalanceList: map[types.ZenonTokenStandard]*big.Int{
						utilZ.TokenStandard: big.NewInt(77213599988800),
						utilQ.TokenStandard: big.NewInt(772135999888000),
					},
				},
			},
		}}

	// Genesis Blocks

	if ctx.Bool(GenesisEZFlag.Name) {
		z := big.NewInt(100000 * constants.Decimals)
		q := big.NewInt(500000 * constants.Decimals)

		utilZ.TotalSupply.Add(utilZ.TotalSupply, z)
		utilQ.TotalSupply.Add(utilQ.TotalSupply, q)
		block := genesis.GenesisBlockConfig{
			Address: localPillar,
			BalanceList: map[types.ZenonTokenStandard]*big.Int{
				utilZ.TokenStandard: z,
				utilQ.TokenStandard: q,
			},
		}
		gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &block)
	}

	if ctx.IsSet(GenesisBlockFlag.Name) {
		input := ctx.StringSlice(GenesisBlockFlag.Name)
		for _, s := range input {

			ss := strings.Split(s, "/")
			a, _ := types.ParseAddress(ss[0])
			z, _ := strconv.ParseInt(ss[1], 10, 64)
			q, _ := strconv.ParseInt(ss[2], 10, 64)
			zs := big.NewInt(z * constants.Decimals)
			qs := big.NewInt(q * constants.Decimals)

			utilZ.TotalSupply.Add(utilZ.TotalSupply, zs)
			utilQ.TotalSupply.Add(utilQ.TotalSupply, qs)
			block := genesis.GenesisBlockConfig{
				Address: a,
				BalanceList: map[types.ZenonTokenStandard]*big.Int{
					utilZ.TokenStandard: zs,
					utilQ.TokenStandard: qs,
				},
			}
			gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &block)

		}
	}

	if ctx.IsSet(GenesisFusionFlag.Name) || ctx.Bool(GenesisEZFlag.Name) {
		plasmaAddress := big.NewInt(0)

		q := big.NewInt(1000 * constants.Decimals)
		fusion := definition.FusionInfo{
			Owner:            localPillar,
			Id:               types.NewHash(localPillar.Bytes()),
			Amount:           q,
			ExpirationHeight: 1,
			Beneficiary:      localPillar,
		}
		gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusion)
		plasmaAddress.Add(plasmaAddress, q)
		utilQ.TotalSupply.Add(utilQ.TotalSupply, q)

		input := ctx.StringSlice(GenesisFusionFlag.Name)
		for _, s := range input {

			ss := strings.Split(s, "/")
			a, _ := types.ParseAddress(ss[0])
			q, _ := strconv.ParseInt(ss[1], 10, 64)
			qsr := big.NewInt(q * constants.Decimals)

			utilQ.TotalSupply.Add(utilQ.TotalSupply, qsr)
			plasmaAddress.Add(plasmaAddress, qsr)
			fusion := definition.FusionInfo{
				Owner:            a,
				Id:               types.NewHash(a.Bytes()),
				Amount:           qsr,
				ExpirationHeight: 1,
				Beneficiary:      a,
			}
			gen.PlasmaConfig.Fusions = append(gen.PlasmaConfig.Fusions, &fusion)
		}
		block := genesis.GenesisBlockConfig{
			Address: types.PlasmaContract,
			BalanceList: map[types.ZenonTokenStandard]*big.Int{
				utilQ.TokenStandard: plasmaAddress,
			},
		}
		gen.GenesisBlocks.Blocks = append(gen.GenesisBlocks.Blocks, &block)

	}

	file, _ := json.MarshalIndent(gen, "", " ")
	_ = os.WriteFile(cfg.GenesisFile, file, 0644)

	return nil
}
