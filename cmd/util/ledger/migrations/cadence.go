package migrations

import (
	_ "embed"

	"github.com/onflow/cadence/migrations/capcons"
	"github.com/onflow/cadence/migrations/statictypes"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/cmd/util/ledger/reporters"
	"github.com/onflow/flow-go/fvm/systemcontracts"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

func NewInterfaceTypeConversionRules(chainID flow.ChainID) StaticTypeMigrationRules {
	systemContracts := systemcontracts.SystemContractsForChain(chainID)

	oldFungibleTokenResolverType, newFungibleTokenResolverType := fungibleTokenResolverRule(systemContracts)

	return StaticTypeMigrationRules{
		oldFungibleTokenResolverType.ID(): newFungibleTokenResolverType,
	}
}

func NewCompositeTypeConversionRules(chainID flow.ChainID) StaticTypeMigrationRules {
	systemContracts := systemcontracts.SystemContractsForChain(chainID)

	oldFungibleTokenVaultCompositeType, newFungibleTokenVaultType := fungibleTokenVaultRule(systemContracts)
	oldNonFungibleTokenNFTCompositeType, newNonFungibleTokenNFTType := nonFungibleTokenNFTRule(systemContracts)

	return StaticTypeMigrationRules{
		oldFungibleTokenVaultCompositeType.ID():  newFungibleTokenVaultType,
		oldNonFungibleTokenNFTCompositeType.ID(): newNonFungibleTokenNFTType,
	}
}

func NewCadence1InterfaceStaticTypeConverter(chainID flow.ChainID) statictypes.InterfaceTypeConverterFunc {
	rules := NewInterfaceTypeConversionRules(chainID)
	return NewStaticTypeMigrator[*interpreter.InterfaceStaticType](rules)
}

func NewCadence1CompositeStaticTypeConverter(chainID flow.ChainID) statictypes.CompositeTypeConverterFunc {
	rules := NewCompositeTypeConversionRules(chainID)
	return NewStaticTypeMigrator[*interpreter.CompositeStaticType](rules)
}

func nonFungibleTokenNFTRule(
	systemContracts *systemcontracts.SystemContracts,
) (
	*interpreter.CompositeStaticType,
	*interpreter.IntersectionStaticType,
) {
	contract := systemContracts.NonFungibleToken

	qualifiedIdentifier := contract.Name + ".NFT"

	location := common.AddressLocation{
		Address: common.Address(contract.Address),
		Name:    contract.Name,
	}

	nftTypeID := location.TypeID(nil, qualifiedIdentifier)

	oldType := &interpreter.CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              nftTypeID,
	}

	newType := &interpreter.IntersectionStaticType{
		Types: []*interpreter.InterfaceStaticType{
			{
				Location:            location,
				QualifiedIdentifier: qualifiedIdentifier,
				TypeID:              nftTypeID,
			},
		},
	}

	return oldType, newType
}

func fungibleTokenVaultRule(
	systemContracts *systemcontracts.SystemContracts,
) (
	*interpreter.CompositeStaticType,
	*interpreter.IntersectionStaticType,
) {
	contract := systemContracts.FungibleToken

	qualifiedIdentifier := contract.Name + ".Vault"

	location := common.AddressLocation{
		Address: common.Address(contract.Address),
		Name:    contract.Name,
	}

	vaultTypeID := location.TypeID(nil, qualifiedIdentifier)

	oldType := &interpreter.CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		TypeID:              vaultTypeID,
	}

	newType := &interpreter.IntersectionStaticType{
		Types: []*interpreter.InterfaceStaticType{
			{
				Location:            location,
				QualifiedIdentifier: qualifiedIdentifier,
				TypeID:              vaultTypeID,
			},
		},
	}

	return oldType, newType
}

func fungibleTokenResolverRule(
	systemContracts *systemcontracts.SystemContracts,
) (
	*interpreter.InterfaceStaticType,
	*interpreter.InterfaceStaticType,
) {
	oldContract := systemContracts.MetadataViews
	newContract := systemContracts.ViewResolver

	oldLocation := common.AddressLocation{
		Address: common.Address(oldContract.Address),
		Name:    oldContract.Name,
	}

	newLocation := common.AddressLocation{
		Address: common.Address(newContract.Address),
		Name:    newContract.Name,
	}

	oldQualifiedIdentifier := oldContract.Name + ".Resolver"
	newQualifiedIdentifier := newContract.Name + ".Resolver"

	oldType := &interpreter.InterfaceStaticType{
		Location:            oldLocation,
		QualifiedIdentifier: oldQualifiedIdentifier,
		TypeID:              oldLocation.TypeID(nil, oldQualifiedIdentifier),
	}

	newType := &interpreter.InterfaceStaticType{
		Location:            newLocation,
		QualifiedIdentifier: newQualifiedIdentifier,
		TypeID:              newLocation.TypeID(nil, newQualifiedIdentifier),
	}

	return oldType, newType
}

type NamedMigration struct {
	Name    string
	Migrate ledger.Migration
}

func NewCadence1ValueMigrations(
	log zerolog.Logger,
	rwf reporters.ReportWriterFactory,
	nWorker int,
	chainID flow.ChainID,
	diffMigrations bool,
	logVerboseDiff bool,
) (migrations []NamedMigration) {

	// Populated by CadenceLinkValueMigrator,
	// used by CadenceCapabilityValueMigrator
	capabilityMapping := &capcons.CapabilityMapping{}

	errorMessageHandler := &errorMessageHandler{}

	// The value migrations are run as account-based migrations,
	// i.e. the migrations are only given the payloads for the account to be migrated.
	// However, the migrations need to be able to get the code for contracts of any account.
	//
	// To achieve this, the contracts are extracted from the payloads once,
	// before the value migrations are run.

	contracts := make(map[common.AddressLocation][]byte, 1000)

	migrations = []NamedMigration{
		{
			Name:    "contracts",
			Migrate: NewContractsExtractionMigration(contracts, log),
		},
	}

	for _, accountBasedMigration := range []*CadenceBaseMigrator{
		NewCadence1ValueMigrator(
			rwf,
			diffMigrations,
			logVerboseDiff,
			errorMessageHandler,
			contracts,
			NewCadence1CompositeStaticTypeConverter(chainID),
			NewCadence1InterfaceStaticTypeConverter(chainID),
		),
		NewCadence1LinkValueMigrator(
			rwf,
			diffMigrations,
			logVerboseDiff,
			errorMessageHandler,
			contracts,
			capabilityMapping,
		),
		NewCadence1CapabilityValueMigrator(
			rwf,
			diffMigrations,
			logVerboseDiff,
			errorMessageHandler,
			contracts,
			capabilityMapping,
		),
	} {
		migrations = append(
			migrations,
			NamedMigration{
				Name: accountBasedMigration.name,
				Migrate: NewAccountBasedMigration(
					log,
					nWorker, []AccountBasedMigration{
						accountBasedMigration,
					},
				),
			},
		)
	}

	return
}

func NewCadence1ContractsMigrations(
	log zerolog.Logger,
	nWorker int,
	chainID flow.ChainID,
	evmContractChange EVMContractChange,
	burnerContractChange BurnerContractChange,
	stagedContracts []StagedContract,
) []NamedMigration {

	systemContractsMigration := NewSystemContractsMigration(
		chainID,
		log,
		SystemContractChangesOptions{
			EVM:    evmContractChange,
			Burner: burnerContractChange,
		},
	)

	stagedContractsMigration := NewStagedContractsMigration(chainID, log).
		WithContractUpdateValidation()

	stagedContractsMigration.RegisterContractUpdates(stagedContracts)

	toAccountBasedMigration := func(migration AccountBasedMigration) ledger.Migration {
		return NewAccountBasedMigration(
			log,
			nWorker,
			[]AccountBasedMigration{
				migration,
			},
		)
	}

	var migrations []NamedMigration

	if burnerContractChange == BurnerContractChangeDeploy {
		migrations = append(
			migrations,
			NamedMigration{
				Name:    "burner-deployment-migration",
				Migrate: NewBurnerDeploymentMigration(chainID, log),
			},
		)
	}

	migrations = append(
		migrations,
		NamedMigration{
			Name:    "system-contracts-update-migration",
			Migrate: toAccountBasedMigration(systemContractsMigration),
		},
		NamedMigration{
			Name:    "staged-contracts-update-migration",
			Migrate: toAccountBasedMigration(stagedContractsMigration),
		},
	)

	return migrations
}

func NewCadence1Migrations(
	log zerolog.Logger,
	rwf reporters.ReportWriterFactory,
	nWorker int,
	chainID flow.ChainID,
	diffMigrations bool,
	logVerboseDiff bool,
	evmContractChange EVMContractChange,
	burnerContractChange BurnerContractChange,
	stagedContracts []StagedContract,
	prune bool,
) []NamedMigration {

	var migrations []NamedMigration

	if prune {
		migration := NewCadence1PruneMigration(chainID, log)
		if migration != nil {
			migrations = append(
				migrations,
				NamedMigration{
					Name:    "prune-migration",
					Migrate: migration,
				},
			)
		}
	}

	migrations = append(
		migrations,
		NewCadence1ContractsMigrations(
			log,
			nWorker,
			chainID,
			evmContractChange,
			burnerContractChange,
			stagedContracts,
		)...,
	)

	migrations = append(
		migrations,
		NewCadence1ValueMigrations(
			log,
			rwf,
			nWorker,
			chainID,
			diffMigrations,
			logVerboseDiff,
		)...,
	)

	return migrations
}
