package blocks

import (
	gethCommon "github.com/onflow/go-ethereum/common"
	"github.com/onflow/go-ethereum/eth/tracers"

	"github.com/onflow/flow-go/fvm/evm/types"
	"github.com/onflow/flow-go/model/flow"
)

// NewBlockContext creates a new block context for the given chain ID and height.
// This is for use in offchain re-execution of transactions.
// It includes special casing for some historical block heights:
//   - On Mainnet and Testnet the block hash list was stuck in a loop of 256 block hashes until fixed.
//     https://github.com/onflow/flow-go/issues/6552
//   - The coinbase address was different on testnet until https://github.com/onflow/flow-evm-gateway/pull/491.
func NewBlockContext(
	chainID flow.ChainID,
	height uint64,
	timestamp uint64,
	getHashByHeight func(uint64) gethCommon.Hash,
	prevRandao gethCommon.Hash,
	tracer *tracers.Tracer,
) (types.BlockContext, error) {

	// coinbase address fix
	miner := types.CoinbaseAddress
	if chainID == flow.Testnet && height < coinbaseAddressChangeEVMHeightTestnet {
		miner = genesisCoinbaseAddressTestnet
	}

	return types.BlockContext{
		ChainID:                types.EVMChainIDFromFlowChainID(chainID),
		BlockNumber:            height,
		BlockTimestamp:         timestamp,
		DirectCallBaseGasUsage: types.DefaultDirectCallBaseGasUsage,
		DirectCallGasPrice:     types.DefaultDirectCallGasPrice,
		GasFeeCollector:        miner,
		GetHashFunc: func(hashHeight uint64) gethCommon.Hash {
			// For block heights greater than or equal to the current,
			// return an empty block hash.
			if hashHeight >= height {
				return gethCommon.Hash{}
			}
			// If the given block height, is more than 256 blocks
			// in the past, return an empty block hash.
			if height-hashHeight > 256 {
				return gethCommon.Hash{}
			}

			hash, ok := UseBlockHashCorrection(chainID, height, hashHeight)
			if ok {
				return hash
			}

			return getHashByHeight(hashHeight)

		},
		Random: prevRandao,
		Tracer: tracer,
	}, nil
}

// UseBlockHashCorrection returns the block hash correction for the given chain ID, EVM height, and queried EVM height, and a boolean indicating whether the correction should be used.
func UseBlockHashCorrection(chainID flow.ChainID, evmHeightOfCurrentBlock uint64, queriedEVMHeight uint64) (gethCommon.Hash, bool) {
	// For testnet & mainnet, we fetch the block hash from the hard-coded
	// array of hashes.
	if chainID == flow.Mainnet && evmHeightOfCurrentBlock < blockHashListFixHCUEVMHeightMainnet {
		return fixedHashes[flow.Mainnet][queriedEVMHeight%256], true
	} else if chainID == flow.Testnet && evmHeightOfCurrentBlock < blockHashListFixHCUEVMHeightTestnet {
		return fixedHashes[flow.Testnet][queriedEVMHeight%256], true
	}
	return gethCommon.Hash{}, false
}

// Testnet52 - Height Coordinated Upgrade 4, Nov 21, 2024
// Flow Block: 94361765 4c9edc817afeaaa6aeb5e63504ed3f5ba8bcbba3931e53f5437d911a1129b431
// PR: https://github.com/onflow/flow-go/pull/6734
const blockHashListFixHCUEVMHeightMainnet = 8357079

// Testnet52 - Height Coordinated Upgrade 4, Nov 20, 2024
// Flow Block: 228025500 7eb808b77f02c3e77c36d57dc893ed63adc5ff6113bb0f4b141bb39e44d634e6
// PR: https://github.com/onflow/flow-go/pull/6734
const blockHashListFixHCUEVMHeightTestnet = 16848829

// Testnet51 - Height Coordinated Upgrade 1
// Flow Block: 212562161 1a520608c5457f228405c4c30fc39c8a0af7cf915fb2ede7ec5ccffc2a000f57
// PR: https://github.com/onflow/flow-go/pull/6380
const coinbaseAddressChangeEVMHeightTestnet = 1385490

var genesisCoinbaseAddressTestnet = types.Address(gethCommon.HexToAddress("0000000000000000000000021169100eecb7c1a6"))

var fixedHashes map[flow.ChainID][256]gethCommon.Hash

// generate the fixed hashes for mainnet and testnet
func generateFixedHashes() {
	mainnetFixedHashes := [256]gethCommon.Hash{}
	testnetFixedHashes := [256]gethCommon.Hash{}

	mainnetHashes := []string{"acb08ca38e1f155f1d038c6d2e1acc0a38915624b5772551c4f985f3ebc3a3e0", "5914c330c16ee5e6b8e60d0435d0390ef3a29cde3e177090c23cf34e111792eb", "89efffebedded274fc0c72a4d3e953d990b5f54b82b696c65390f87b2f9b331c", "824a13a4d2252ff045cc785aa77c6ab8f85b48a24aa5ac198417bc05248e3d20", "5c0eefa82e36a4a7bc8b67f4856f756407189f4011d74c1cc6125599bcd6a18d", "6e6435cf4a9dc503a213fee4c8e5909f32ef284b3dbe780fadf78ce2a70a6a56", "f312e8571dbd7e2c347a0032d7ac42b62966a833ddacf2ba1fd1b0c1dbf755c0", "e9ef75691eadf0e6e9ca88cc0dc9c29e520aa611dc21ba390eead58284949873", "bc2fedff2ca293a75dc36c577dd05742671549586a333e458c8c723a3d3ba40e", "04256e11dc4ecc63eee1b3ad22e31860d26a1cc2103e34f91f12f4a61cd3150f", "454081c5e315537eda05e5fdd8e5b34df7473386c16d140dcf0df9c35159310c", "f4a897310404d46b19a87a45f4e53743c12c1b4530383d3a8ecc972940461cf0", "81765ca144baff8e65ebe989403c8f86ede26cee5580ff5320817a108e54e887", "cb96415a6f8d3ac6abed34fcc83b2745775c0bdffb7561392e1eeab63c28bd19", "c70a0e0279c46e6fae534bf1dbee7796078ae1a9c214d7719b12dcfd4fbbf55c", "4668064ef2d42bbae07e44276b55922ff7830f8ff203d91f91854252bf42dcfc", "bc966f7acce679568f84c0f6164079a4b238b856bce15091fd62d3d94506b92e", "b6d1beb5b1be5715eb61f0b6528e339c75604f2ebf0605238905a0c1fc4f0594", "e367fba588c1fa71fec1e769963d3106a0e73d13e2ec441d2de44949673804b0", "2ea9607ed6160325c0fb6514ad2d0eb4397afe131c787a6d189e39907ade71ac", "5297cff89b9f573c2f7920be0c8d3e71c32f3016a1c893e9f41048e979533a70", "9f81c00947b14ddfb6793437a787d0bb8ba5692e264f7f5087bbd4e8bdc961f7", "6ef10778647ef844ce9c53b740890980a564619e3ce866faea7bd75b67971873", "db1d873cfb81b4aa32b7d575542d545176782737d7a7f3c9a288205124e91467", "cdab6dc09455023a24c38ae89232d4dd5e76d13935f38eb4d60a8ae3c2f87270", "9cc982be62369ee866334c1ce1660046cf23f109e7baea6a6525ac9fa2657143", "8ef919c45b46bbee779b7511a0dfe23cc9631bbd48103e2f9b7cbe9b492ac61e", "26da1293ebe6664711039a56d9f2fcb245a7b3803c4219cbd27430307624977b", "681c1001f30ebd45fe2ec071e168476c3d3367a18cbbb14784f5ad549b6f6c76", "03a921c3db624982c82090c5f562599f0bef0e542bf145c3784e055dafc43f75", "e0304d9cd962ba44165e5dcd4c29bae6e9eeaa263413c8e0ca41d0cf883130a9", "a931939f13b5dc447464b115b487820467743016fb5bee8b43864ad836a98261", "c7ed304fca9a13944e3f81c5fe4e8df4a7327d1c89fd63de357551e99996d9bd", "80f7f4870cdd284f129babe4d08c002acb3a63a43b6d73ea39c28820c9002d20", "e2d09b3b3d27e1d5448c1772e9d300fa1615c791e2d8b4ebce3d4c24e6058cbf", "754869cba21c3bd12d94a23571a947f8a33dc369e6cf4ca18f2fd40e7b5f5a53", "e2dc7e12450ddbc50087fd76041623416c13010be0a804f4676af1768e5d61ac", "7bb9175b93b7cab1e02a3f571f399db950908b57b45f1d584f3a5ac1781496a8", "2e7e5f02e2c107b73dae8b20a0f52d52ea1aa7248c6b4876f70808e5168dada9", "e19d12c9f01d7b8cdf6831e269e0a011cd744a10aa1da65780f80a50c812eafb", "6bdaa6249d9616d1244a7e23995dc416b9f3cc464ca2d5941cccb8d5b1a1eac8", "38e68d98e93683c14c3c0cbf82298c329857503bd88e488c6cc8ce83436295bd", "e6149e3ed7747619bcba88919daf4e9dc167a276887e8bad88158fe796aff9a9", "e6c8562da3023e8d864f04545f26ec68f5a3d9ad984e225104ee63617e22cdec", "677b31d0b0fd1be96e13968e9cf679797082b5fe7a2c3a9934516f9d05a35c5d", "d894c76d4e18fdd1424435028631b008224fa480faf7dd4404faa26805f226f4", "38421bae5c3e39fb73178621e97fc5a8eeb1b6e25a8faba76aa6f808819aa263", "ac90729f29643e4107280ae8b69fe4e862c1cfbeef57a9f70579b16db9479410", "a671eff0c687d04816889d0c56e704796690771cb5a1eff8c83ae5ef417f79bf", "601fc2b0ca9979c4b092a812a88934f46381e42037278523f191e8207ad1e20b", "0dcadeeb37a0836c4aa287c43a7f1e6e72eaabc8fb0f5ad6209f87e69f2bf412", "02bd187372fe4c6bf894fabf7191afca7f2f052e5d42a2cc6fb7f2e6b1a07716", "39da57b24b312b1838a44de3286c51a0189608bfaa5904a7a2a35c675b947322", "fe16a19cdeacc8ce05bf38d3617c1e90579b6991775d3c0358bf3dc52aeae032", "9e7e8957797b6fb78679c60c249cb8b83e045e760a6ec24c506d565ae94c1730", "7ae42245a1611e7d32d573ddc2863f9f1902683a17d46b57794ec90ad61a9d6d", "f599ba650e87dcf51733485aef08e71f1d8f5e47c459250902daf3db9bb9f526", "7d914de318e12963c059aa04fe03cb45849b16620a1e7c2a883164bca65ad3e7", "d66014e30f72e1bb309235b6d9b8db6f6fe13b624c0ec81ed657ef215d13e29f", "7d25f2ab344c3ce12cad30a992264dae73668e694d8690bff158c0c66951b6eb", "c4eeb03288ac4ec6166d31909e3fdea938b76f637bdd4963910aa0cfedf4496f", "a30beb208f4ccaec67f83f72b72d18a428b5876ebf53184248ab3438c226729c", "67d9c883f3f8df5afdd8e897f54f4ddd4522d761d23429c1697034e3bebe8df6", "fffc4c5760e75dc839acb6654e0274afbe07b20ca46e90b0da0262432473955c", "3238927e1ff0d18a573cff7ea6d081579bd9ec9de9f6ba2f67301ef88d65057c", "3af6b7b1124dbabca4aa2734711484ff6fc6d76130cf81633e051ffdc01b3583", "0475c59145cad6563ed3f0cae8d03a09c73d4862c5a090f8d5ba5c43f3f744fe", "896c5230f74946f18dc31d879d228715303ddaf01d6c1050dc4cac1cab8f5273", "a0959444effc54fc3d04a31a87ec466063c510762b2b4e099cda3794c0d59c07", "0f7b8362a5f8bfe9104a2dbcbf25dac6dfaae4fd41862cb0f0e028062b7db9fd", "83303d47daa193a0e9f1cb38b7fef57508b6f8f80aa46d5663f64800c9bd25de", "82892728f36bf81b17e2fc6762444d938f5b8b6e80c09c7189e73a8b6b9b2b04", "39f93ea2ce0afb9ea531662a38cd65984ae38b10076a37ddd10fd45ed35674d0", "6783668b699abfe0b3bcbbc79988e7c1c5191038497cd73e52502702d18b8cdd", "6fb5147a8b6cff70490dd2dcbee8c26e32808034978f8989bd0d73ac1c5cd79b", "b3b550f194004cfb54f24f738fe901058fa4eea680d5f704a1e49996aa7df019", "2e81de6c3c6a1828322fea4a5d7add9c9c4bc940d37760cd78a15f7185bbec5e", "8ea70bc7c983074e7d32d9c47e24ebea9ddc0a04aa4061e82c40566cbb886061", "f0c2785d27868755124be6ae2cdde18804893493f53c2bf3b9ce3bc37a983afb", "f7e684111cb2b43644b5e2a07bcdfbb9231ba8647dae01103bb15ee84ed59dce", "a4c839a3ec06907bc87e18a06aab314e184bf55d8d66acf57012d81bed5f5a0a", "e6f94c1f935b7505d65b70571a169572a5586582dfcd7ec43614eb5a53169556", "5f67958fb79aec5be7e950deb0a9a86fbcd328eb75298946014f06c200fd8dcb", "ba54fe0ea8a35e899ec5e5ff9aa888ae7c5ed8630336a098c131809b6e3a815c", "11eea2b61707439bfcc198d3d483fc7fbc8f5c83f70b6190b6bd1bd11a0edfd4", "d1088b19e8814dca954f5a78f827ea6e20de2b8e0839d5f2d2ece9cc58d72c76", "c5e0a35346bc8c9a45338f5844cd13f5f5b94ae90494c8609b7fe2dd69925429", "32089be74f3bd2191d7e8116a742b40f613d75bd77765c28a11d937755ca52fe", "f584db14565d9abc212b02935724bf05da840670b46e83a64990d7463571f9c2", "a959dab01d61bfb54bfcb58bcdb609f42e2a062fcc63eb3d5b866e582fc589a8", "cf09a5617dde47025acdc7dc544f9d78fa396c383ecee103b5b74b532d9a586a", "03bbdbf7f22cd92a696f1ffac34c99d7e57c201edfdfc617826ef2f648d38475", "7f1a8c24c456052fcc3721707a56c457eb7d80ce8d83d8d23c5a9a0cb70eeaec", "2ed7147f47b4f12924358d18f778cec7d28dd53e9189a5096a7449f42a1ce29d", "c78b588dc0e967fc85abad5d5a18f2be86b7a77363ce701f245507a7043de3f2", "bcc0b4ed36d1512825bb2a2db5ed41cf5a7f5fa5634c8199415eae7a145ab772", "c520f97ac043cf2431641d4532c4a44f9664e728c08382ec798ac49997f19695", "a7825ca8bc2f6ac8556b88cc9a3c2a533504e5a8e011149cc15eaace9320c23d", "de8cc99029674ccd55105e8b5182b22e8c219a8a35e9e5fbb386d232e8e1ffd4", "2ae0a0239db0cdc5108215d38f30d783b4619824a5b420cbfca4fd6242586fa5", "3ce22aea444053e3456ca4edacb1060a5a355a7ca7e585af873388f99e654028", "f18aea7d73a0a8b2c313eaf7e742a08225e68341de787a4003fe49c06a5d4d13", "82dfb93809f99c59f6d41402e863580fe080278faca77cf2eddb651fedb77b05", "08c1d039a238c625ff6715aafc33ee8a1675bfa482ba6edbd0d9cc63d947b5b5", "a90aa55518cc9000eceade9b79644cf723c22f60caf849604dbb2ac22a8b5a86", "077cd67222a27be3640cd4d5cb3946bbc0f7df3fca7c5dd2ab66e4f9187f979e", "e8ccaf643c060c92fca26ab6adec347a6c3fee28bfb2089c5973bfc319cd8da5", "49e9c991a4d793b5ab62c3dc16290cfee8389fb12ab90e182964dbbefb72ad3e", "9ab6a29e6b5cc88f1791de37ab48ea5daa9e222365fb2b590c8f04109a372a5c", "8022544ade8da7ab8b34bf3bc8cd15e90697a4e72e760c809880830e2aadbdc5", "cc6b301ae355cdadf19e48e6cde96d98961c9aee896bf9ed815cb47dae0e1c22", "02b5781e6a697fdd26883f63ecb7d947e0789ed1fffb551bec429a139c0dcef8", "bfd97e8342aee5eee212638e37b691edac398b54bc535ae3458ca72c530ebba6", "1546f5334900491745f87d82fec8082c65cbf6e975b9474041cc7e22fe369130", "8d42b170698fc0c2662a2fc6d1017a45bb7af9c335e1c5a2cc107759d3aab7dd", "0cd379d9866856ed9fc3ad93190cee5d5ab8dc738b71fbb4bfa14e44d2b342bd", "b287e14cd59493d0f0a8d6a8a8ccb056da71527af9610bd38a80f89f43ba9e0e", "699f32bab442ea206544c0478fbf8e55093bdf246346014f242bdf1be60b9b9f", "ef55d74e0c1b660dd69bbef8b1d87ad827da29f2ab5169c14fd17e5ab3f2906e", "cef5074e106ea292c52651ae438bd80ff34b8ebfd31e00ab137190ac8829967f", "312dba438b767fb62feeed74223e5345241e3c9d078863b82c9768d52635d6e2", "102da56cd23259629a60db3c3e60eb2ddcf124ee47ed6e37e09b1cbd023a2a55", "d8a8e6bd81976810315c0950dffc466ae9ef5440629cbfdae970adf9be85a2eb", "d890f76eed51ff1f08b2e08c13123b5b59b92db93874c3a1774c22589ccdfbfa", "8b9a63cd3ff092638e11ed29b542cce0b5098f2f3ef965f5c0b4c18cae90bc69", "1abf154ec1d34306d97189ea9af96a6a33c4bdc597cbc14897b89decbf38661e", "07ca0710e82029b6385832a4b546e0336c587b8ea9280fc384afb611d80ab7ba", "b043f239fe9bb9e78e4102b7ed49d35beea61ed7d677eb53cdbcbbc2783b4079", "1b50849c36638c9afe17cb095d4bc978d8883404b1c58cd3acf2ed09f188c602", "51853c7a3fa6b70dde4f16610ab43241a89ebd3bcd0c473606833551358a8f7c", "350b1984c35d9d48f6f0dbf97e33e76edabb0125538b52927182ea00a4736021", "126ea9840493f9ecbdb8cf04327f0ab8c9315a7420772b2bfd263fd16d1e28b1", "b0190ad8ed68d4f8f91a54240ac7e2205be58b5f8ac5f23e8ffd280c3e554c96", "47e46e9f19a2088625dcb5a1a5c6210f3b4f30e748ba23c6391f314fea4f5bcb", "82b39c4162a1e38739942ca62fb80aa1de7f9a833c0de58d67796a243175b917", "d0cd963ed709c3573789cd8e4c35ae28692db1a6c99f7b38aceb7411a4f7be98", "37a325b033d3f6f1d56f27dd4c5169301f7eba32e8f4c8a8349cc7ece87ddd9d", "51c95c79b6819aa2efb727fd29cd73368488e828fbde2b64af4576e79bf242f6", "3bc469e4ad8a997d246006f09febb05acfe065db25c4a33c8f2437b0dfef0878", "b58599ffd76d2147235706a200780a5ec6195e2a5c13d2b7b8d242b7c1958d16", "140462e616516eb56075d1ea6c01661c2f2638e471a28ccbfcb5d5cf94eb3e74", "9d1cd56f1a33c62840af5b75f5b1e3b0a1475db362a7b8999b8897c8defe8579", "5adea11dd63543557d0f95028656284e482e894672342b664c2d483654c96271", "b01f5826fe1aaf8cfa9955d9f2d66fa2e896e8406117b87291a05c8c0b1510d2", "5974e67c55df5f4c6a0e3230d4322aa70ee9ff975a6e0c65b4fdcf6b84d4b31e", "329fdcec3d7c1e61b190fa5ab4c6d58cdce2441671c695470c95e00679390289", "a5f0189e64f96ef6e06f5208718ba903d1934eb7f0b85aa38fade6e45e1278bb", "0b4ab1c1a890edd1b714c390399293cf1e1d1deef68ee4ef005e3b68ff17ce6d", "5449ba71016c81101f874c61702fe7c472d50c2bff7c815028cc6c84d761aaf9", "d5d5fdd27c59b705652ba82caf7ec3ddd07d4e3168ec4006b3c21b431cf82971", "8a2e4c552b152c8b76cb7e07ea727f26c607b600bb382af4b9f066041156a7fe", "96a49a267355918ba085c665fefbcc6a53e29b35734ec8d1570cbbec61081154", "f040e21168602b67d8afba7aff7cf0aaa4acdb463aedab7a29fe2248f41582d3", "a83dc07b7b05d05954aebd19afb76ab9794e35d1f0bfeb0222f7434579a9fec8", "3259f7323e6e0a7ca95dbec594b4b7ce5f7350bd54ac97a3bdc35c333e1024f2", "c84287dce56c2837eb140485775c13645d3d7195bad44174497c1624e3d6bcf8", "5326aed27fbdb6a4e59a974bee60aa1ae71195aaa311bfeba212c152e0f56266", "8d83acde8c0c2606bbda85fef834620309546855d5917d6162a3f14683095b47", "cba4417044bed9ff8f494919f23661efce69821367fe850a837f7cdd64f5d814", "1bf83c9a48b54e8b4b095bee90f5bcc1ac8e8897b351d93205a64c133bc5bd7b", "0ebb774b03cadda941343d9b2bbf2e7075f049e6e309dd232cf44a36578935ad", "d193e2601554fb3d1fa0c638e147297a76e4a6ac2c02209bc65d7294dbf002e6", "a9b3ba41d99da589a8dd1dfd776d121e6d4ac4f1ee52d1cc3517d2226fc09ad9", "dd53cbd732125e3f22ef9fadd789685d10a49f88f21a6dd66c3790a4b7f2b85b", "6f827b1068f38167235778d893da3e6c7a949a6641fa5b0aa4a116449e7545ba", "80c4debfeb8d3433350b12856003a0378485b087a0e51d4a974ec88fe8b899b2", "addf88642352377a5d80a9f576e1ed7b8754c09aba6be508e2b8f3b1d7d9e042", "8c961cd106e03576e181925fa16dfda42302f96da8679ef61eb64c1a4742e5b6", "7c02dcbda0fa59f3e843836105151bc1a49a66e2a02fb5941595d23abdd376c6", "45da6f88684c89476755a45df16d1bb602fde60f95d8756311495bb53b441637", "3df1b14731bb4b7a070864eeade24fa37c3584475fe3cf199f41709710ac7f4a", "6638900a817ceda30dbfcc8931ab64d047b281c71ce9e7d203f8790fcea042b1", "b2378c5c9b4812924571836703eeae38364924c2c0430e0a671f2b3a8d338130", "f4825a9397baa4bf07ad69e8dc7e69c03a76c0d394160729542f1b46ff03f338", "50573280946a2c75b36064277f4bbb79875881c6f9f55dc834b0f408ce02be00", "3a6903db22957442e3bd81727d3038c69562403aa8584302f49c28e5f0f4f5ce", "081be91f15adc3c6591e317a188d524c1d16d01ba396508e5ed6a897c169e9a8", "84bddbda2880e71a37578cd427c7602c3580b6af74fe9640cdad994678ed6edc", "c1b6f2cf31192cf7a3643b57fa98ee056e0dd6c6f28eec65821f4fb5b6721971", "f9d11cea4b504a360c0d62c3d908d35f5742112588f2a9fa7eefb5d90c1383f5", "478adc2d34dce7af32071a0e2eedb8c7fb6ebb90bfa404f6ebe10776badf1fbd", "8d809a7afb8b0f327646e1efa6f00670642ba9dde2fa2569d67e5c11a2c822da", "0231b304c4325ac717cce997b2f33f885523062f931d812253035916abfb8e47", "f49b278ef762922930de0e7d4b8ada81b64d010539dbf5a2530e1f88c4a6ad29", "617f5ec465f421abd0e6291b6ca5f8e027f2d500b406d87b6056101bac98a1b4", "1081fddf73cb61f080a9fcef1d3ee2bdf466c3ed35876ee82482c1a49bdf2385", "25b819d32eb42de93e50bfbb656030051d7e4ff20d3c78e11506df28a64707ae", "97f38910f204943718d61a88cc539a3f281d540477b0fb2c7929aada1061a1aa", "bf46882478c2a7955093126c7072d7b7fe472967979de522c2c14739bbab7d07", "31a8a2038327e176933240df416d3035861e959eac4528560ff348347c716f27", "d827a95da4a08258897313e839a9613c62de031517db363580c29ccfccfaacc8", "b5de63a660dae61c272f9dc1e646da96eca8a62ef3764c2e3b0ae6b258532268", "60d8f10911e03d48eb7274864a09b19756096e0c28f5ca42a26c4f9b3b7fdc5c", "e5bfc9d179f5fb0810cacbed185cc2b2042b774b95dde4048e8c9b4b4043bd31", "c061bf4ed829c8a43e2c5aa336c67cb4e22635c8e15791cf67ab92e0efb73d30", "aea5b83e75a1dd4f705ef09097965dcd010806537361e228cbe275d783d03a6c", "61fab563337233435da3d3be1e8c0d2332edcbd5bb7085c931e5ed4de2f80ed4", "83044467ce97ee203e81fedac56db84ca469ecf40d278d6e18380db17a719cb6", "fc1dfdc26e01d3974267abd90281f512a6497cea25c198e79318c49a069987f6", "2190499382ade5b6211f7cb7ee8301140c25a8a1e9f95f78a253dd0cee72a9b3", "cdc317b64a7c7d6146d3e63d295b690cea5c8c5deed5e42b094361dcf2038614", "8496b471f706842289855bd5dad8e8ce5a45a0244a537407a62ae82bf28f283e", "dd68dde67735cf4fac77a75f658c01f30b3dd373b7443597c93cf1ee9e1c375c", "7d9fc45eb9727f3a1bc09abb274a904bd1c7c4a8b0ddc131a66d0c35fab12c6b", "d3212e0196e6716a17f83983cfd28a90d4ffd7e7aeb93659a85cc5585266d153", "529b13f078978955ed8c139326647f68298aad6515c978fd532d67814d68a819", "047170f4b389cb5ea020d89957aa1c263d00c7e5923c357fafe2a9539295a70a", "a78a5b14dcf7d45dc1147f12138a46aea7d74643f150947184121c4d8e83aacc", "5fc7cd475121963671bda69d4e83b5da3b915f94780f9b21ad11e14876e6a2ae", "ddf9d7f5b52966e8dc5643c2c7780ce8d5512b581859fad0f11d7862b9082a0e", "98d4c1b60953deba57b070f6686ad1e56dafabe4e0461ff823f7e4f1e2d68a6d", "da05a4b3332528d56f466d3eda964682bc31f90795155ad306960e85239d1570", "52da74b3f44371219361d635f8ec93f428b068aee1d49adfda3f1080b812c403", "03d5d11bb421694cf5829985b2d2ed69cdb66c59874e772f9133feba146e56fd", "95112eaea86e4518c06e90875d56fa96d2c2e1d279263b8aeb55e2ef609c0015", "7385b128fcd181847ccd65e61535a3b1e6c935085feb1f116d07b69f754797c9", "025829df5b0e89d33e50e4da9cbac3699faf423a17a01f82abb1dc5a4aeaf7bf", "b8d71572694b145ff3a891e14463c46bfc2a7f3ce66f4b72489dade529fede9c", "67106f52b3bebaf6148ca60c81bc8802050f299d8e3139a8045ef34a0ee8a83b", "c1e4c64335250f030a8dff08151d8631de4f1737973ece0a66ce5819a6bcdab9", "15ccfb66ee051bc937c87c622ffc726f5f6c9b2c83acf52ed0dc6c63d33e0764", "dabbdafa2406d76784fd51b3f5f4014f97e91a0293e96cac0d7252400793352d", "4c6fe6506950104f209a64e0975ced68826c9d6d5c604725c7cc38119741fe1e", "4c0da75b314859992796ac6fa932c9804e6cbc0372b8af03dc17ee487dd46a01", "126d57ea0faa1410e2bff97a97dee4bb95f931c65e424936a3c663136cf44b28", "7b2000fbbbcb50649b57f7de2fe8e0c2384c16839def35e4ca3b368306c737aa", "191a431907c471085ce9133b62f3ab70ad7ba440ac70790400981e68f46a3a34", "7c6b5159af1596f1b1116915f58686bf5943222da9e864f415626328ee0ae8f0", "c01fc7330f29cdc41647dc85b357fe1c734410628077db6c61f736f2288e91be", "c1c9811dc7c62642ae25fbecdbd276124bbb0b2b3ccde483d81831a092fe8940", "183760186863265934b5678d6701d33b02427f0260de63ab92620cdd0ea0a193", "91036fe1c4780fc9a73005bd4fd0e674d0fdd2c372c1ea036e03d89296322b08", "279f655e7eb78b83a915ebf71097429c2ce71ade9c0ef44f5342f7361dda1c1e", "5b9ea6fe50b0bc7338a425931d5587e7bf29ddc886f95a013dc265f9ad4e6a5f", "e58b9814df7395a036222c5154c090e1edb7413d786f744bc71d3a3e7d3ae51a", "72f05a38389a396e7e099943e7626432809e8fea44b2b59c7f5b1be6e544c477", "66efc642ef86130ae927b9a8211a7898a1a0d4633d800b069b8a435f38a87f2d", "57e2163c10bc4cf0291a22e157e30e2f3bd32774777d562d66b5a56785af16cd", "d8bb29af4ab87ee4c6a5f906da83b486b0cb68804d46520402560fd361f9c046", "08c384948e4a5437238b38307ef1433aad79196ccf3192061381fbe1cb2f95a6", "4961223a92ed9aac5200710c1fac16222cebf4f45d71f9bcf747772ebcc10624", "51749e1822fdbd6e3160abdeae195e281affc52170d4d350b3f205f742ed7b13", "14e8dc225152adf94b64a266a412317eb84fd518055718d4f8261e0fdf8a9826", "4c5dec521f84e603ac86babbe7763fe82125a9eaaa705d8cddd6eec95953a4b5", "8acce8dfac2236fafc944be02d072bfb63ddaea49045e31283d73ab38823fcb7", "12cddcbe68b1fabd5650ded7d323b80460ee122c96e3b58c8b5d29a17b917ec3", "d86759a0c43a2fde5e79adaaa167f9d05338aa8b2bc6fc5f9b1263164aa60343", "5267ab3dd6d646eb7bb1c04b9c23fa104287011a46714accc33f608d36d0f2e7", "d8d8d61f18ebffb56574b089b975016513abba64f68fe0da8c0f8d0a62e0416c", "b0f64d75d6754023267a8bd9dcdd975002ce1aea4d2e8103edf80ed391be3782", "b72d60462ce989b717868769b43678b933f239f977e22e2a0d61fa59721ee3a0", "be9a8aa7883625a2f43670b961827cb4d58edf21618af86e376abb6d743a54c0", "a233d9c85d895c54f9df1c93659ac3b1ad9f46458142a5310f40f11ee9bf6316", "75ee0e41d376721a8a59c7c9dd40282780a0ca863db78dee7a589cfc4c98b3e9", "8b34745c1c95a176ca7f21bd1350ab491763379a3ff99f60214003217f6a7118", "75e4c59a6469d9da7de866054c21689625786d6ced18cf6130aec6fd45766025"}
	testnetHashes := []string{"fa857cb5d4b774e975d149a91dc47687ab6400301bba7fab1a70e82bd57ab33b", "57c87eeb449e976020fb60b3366b867ffc9d88ec5c0f10171af4c7c771462130", "1af58b777b8054a15f3e0c60ec1c0501bd7626003a4fabb2017e16f1f4f9b0aa", "155eb38e56a75c59863434446071a29df399e0b79a0f7627f3c0def08c0dee4a", "fc541a457aaacc00c4bbf2ddd296c212c7c7436a1b15fdf40971436f4679060b", "22461d010d68d2b67a7a3373782af7f75eb240a845c4b1fa1c399c48f7d3eaad", "e62881132d705937c2a0f88cd0e94f595e922e752f5a3225ecbb4e4f91f242e5", "61084954ebe8d12d9ac71a9ce32f2f72c5ab819ab3382215e0122b98ef98bf6d", "b65786186ff332a66cf502565101ed3fdf0a005d8ea847829a909cafc948cdb7", "6ec4b77f75ce5bd028a22f88049d856dbf83b34480f24eb13ed567de839e06f9", "c1db0ccc2f546863cda1e14da73d951e4fa4c788427f13500a1a7557709de271", "b8a6f83f59913bece208fbc481bdf8a0ac332433f8cb01a3c5c1b7ae377f2700", "9a8c588bb81d8c622b8c6d9073233c176440da4dce49433b56398c30239cfe8d", "a84205d415780ed3c0566f9f4578efeb6ec4ca51f8a93cc7f89a00ccce8dcb39", "c5d6591d91eef2ca446351e95dc4134438360c1b7389d975d636cbacba435280", "7be74dcff396c8abf98c6727659575a5b157c9ec98c6f1c9504732054f09aaf5", "a7dcad11df6d5778824decb3624953440a2e8f01036083c10adb36b4465ee14a", "ac6e904295a3d736e7f22ecb5698c1fd8964e3f0afc07ee2487e63ee606b9bbf", "d7c2cff7f8a08373b8aed134fe1fa80899ddaaa8dd7722fca9b2954228b25803", "580cf925b0d2ec1617e17f0be43402381d537e789bd5a08c3a681dcdcae2d731", "c71cde092dddca890f9f44567a651434a801119dfca6fa6a8ad6daa26ce4d6a2", "b010526b4edd19af408eae841184d97f1ad6e8955c4a6ac8240e32f75a26e5f9", "9278a4d8204e7b937c41c71b9f03c97c49203d4cc6e4e6d429be80ff1d11bf02", "d57366198709ee6be52ea72cb54cfb6282ddd6708e487839f74b93c06c9a994a", "1d17a3f34d23425ad6fa3b1f57cb1276d988c3064c727995cd6966af22323830", "660a0a66a46fae20c0a4f2b1a5f11c246ce39bc1338f641ea304cf2dc9bd0940", "e4562f14b6464d2ee4e92764b6126fab3b37b12c8b0ccb0cbc539a0f1d54318f", "3ed39df06d960213a978379790386ec1c6df288a524c9bc11dbc869d1133e86d", "f09abfcf424b6bcb7a54fc613828e5ff756b619c957c51457d833efbbfd9c601", "58b6fe973b269639c2a6dc768e1f1f328c3c1d098b6ded3511b1f8e3393f8344", "398fd65258285061025e5b53043496832acca2a6b61906046605df18767a9da3", "b933d1d819cdbeff8e3acf9cba0fe7b3e6db3bb582da027a0f1e432219bd6033", "99baada49d56352f2e221cf62116c70485a83c1174bcd50cf5ba62b35d1661a9", "19a47884389d1f995a37c7e2b19525d44a27a32a5df2c0b9c2954fe458655baf", "3820ea36958821d31b8f2eee80fc17e72dcf361f052c0399931ce979e9a10293", "3a3655c7bb4fb1814002b468d63f72c0626d4c7df4ceac28a68c970a3686712a", "bc181caec490ade2d715e7d0c82cb9ad3fd685dc962d8ffca00861d88f5366b4", "da92ccf74d37b40738c41222cee137c149889966c54d62d91472d2ea81be37f2", "e51d0d81a40598e0d6281d2bbc56a1d8c5aa3c8233f2bd9be2316ad6a24a2dc3", "6cb2e1aea92658471cc40ec0a4bfd64d8e76bc0b9bb5707306fe89d93158e7c4", "e4bb2e67f5ff721ecfac0df301bf3db9704d47a9d33c2f952be17dc23a113c45", "7d29bf4f9796573cf5274900ec667bced39cb0377409d281a2dbceaf99ec8fd9", "45b32bbc856daf25ad81206623f8a7fb53f0afbb488f72ffef4d8f0a9431e62b", "b5aca33f4af1f65d9e9e35035597b58896d99abd5b7954593ffc70c86a90c94d", "7a21bc1136bd1b288fb5be1fd43b39cdfeae9b424e3da274e241dbc1ac780d72", "95bd53bea9d44609b8b24ff5c30feb08c91d92f239632f8093fbb8f37a704112", "61551f4fb10bd3b97870af25c6c18d8582d6badef8e87e3c5297befd1331003a", "ba43a4bd43dcdf44ce163b58d35df3def39f2a2ed29cfcf76f3d7571827b8bc1", "329c277c2f0555d33e294377bf906c404a163ee653d0894661714a25b1d3c8fb", "6e143a6cf96b0b8eb695bd77b1e28f2a61f4dac8a47b3cf2b69d6737d8441242", "991bc0911f5914677f4ba476717a53b0b889b91cf178ae66c0625167f7ac0801", "541fb4e3a4fc928a017bdce01393ea8113b2236dafbb3809973f7b8352442d32", "9c9181ad53d6506666187974b6b9e3a9c0bee8d085d10cc79f50bfb4248ca129", "1cb89bb5668ac284574be9118a78d3fa5d674c84579c75d5596a47d2acce29f6", "116b1c4d1a8fef4cd852a8841b689fff4f1df3a0f5bbeb545942150f4b806646", "b54f3b2b235b816bda74453e228378fcf9b79a293534aac71dbfeb6b0ee1ecad", "9acb23972960f0b4c5d3c6b061a2a1c4af4f7a6d4a0cdd8ec7134ae7bd59f95d", "17f3d6c720bc5efd5ee8226d353d1b347828e621400a2a282a190f5b7bbdd0f0", "1838dc6001bb37cff89aa8675ec0ae8efdfd35c5dc8a793538c31d08df4b8232", "ad362ed3de8ac036d4a89d31282f26e10cb50fa900c6ad76f7ab06cb7155d234", "2bd6a5464607a39d0bcdd07e15d4752d1a52b644bf9a81d8d7e5f9cff0af30af", "44124bbba59755b9004d53c3e721820c40c1cc163b7639b4c1a03ce6955e292b", "f19520a13533371cea4cc20daeef421c31c0a88d4604e58b56ebef82288cdaaf", "c1796053a6e8847cf3d8a545670dd953d1273dd3d9a6e4df6e59e33950cc2890", "49aeb76ef737a04fe91c3a61dc8c7b87adad5978d8951f8d033ddeae6fa2b720", "bed2427fb70a9a9a576528569ccfd8fc86ab0ecd4ca7a932d5a8f39316f887a0", "a8da98fa12885b4165f7635906d9bc240c2eaa66079bf18f496dbecb68c7c49e", "cd7e523f67b5ab520d1c8972f78db9a8d283c66ccf000aa31cda8216fe2e508b", "1e29c627ce7b6402eb5115c59a48d561f4420c44748d7de2ed185142beab4a29", "5ddf101e94858f06934c6019eaa22b93d88eca16592720e9dcd982894ac27060", "c408705873fb0ab3fc4f5811e69ee20b0a1600f52bb4663e29362f4391601ebe", "ddf70a2c37e60622148124c22f8f0e96b4eba0af4d5b8b18015d574f33923a7e", "d6e1f406e0d96c486c1bcbb09768ff0e5577f18c97cdf2c3e86dda54b4007448", "656b861ba19271a6591c7468af61a9d29e331eccc9e526a3d25517d29bd69809", "24372783456ac149b4fd0dc41ee16d55500a3c433fc3b1bd3c1c45c8a93c89c5", "2bbbb4392ab7f1fd8a160a80163b69b5f8db16fdf97c2d8ee9e29df1d9ebd9fe", "cc9fd404792808740bdee891c8e93e3d41bfe56c2438396d1ca8a692dd5fb990", "38080ff661e3142133b82633be87af6db2d33f386d05f8439672a1984aa88d13", "22b7125bf763c17087306776783ab6d1c50084e8a7435b015207f99295aa1af9", "570c31b148e5f909873e8d2253401a64eace826993948cd2f3f4d03a798c6c54", "f0cb29da50bff805a3a1736dbe33ea139893534d0e25a98f354aa5f279adbc97", "cd6b07cee12ae00058b20a6d31173c934933e6339a00885554ccefde008b12e3", "323fa87c41960355883ada3b85bbc13303d8202761ea70d015841060c7f7fde7", "01c7c87db4a01af781695e2984e68b72f04a0f7859749bcfdcbee73466bf0990", "a79003be6397a1fac1d183ebd14d72f69cfd9ab310cd8f9cc9c3d835b05d7556", "50dcfbe053447768b56f6c3159cc6d37aa5791d87abfad32b2952e36de8a20c7", "21647bb0680b8b09b357a54518a50d6c4163d78889f26ef48bc93cfe43acb16d", "96dfe03bc8aa7dd74ef98b4cb7cad866c851b8fd145f4b5bdb54c7b799e58adf", "87037ff5508a2a31c62cbef1feb19f3ec22f44ade292e0a036e8a7d8ef3d13bd", "6e7336d4e63a744ae45cfd320ca237ba4b194d930bcbfbfde2d172616df367b8", "780126f3f77af11cac4a71371812160e436d50f09ee01eb312d6839b7dd4e3a3", "9373a2bdc426bc5bf3242c7f3ecc83a19f2cfc0772ecdeb846e423fc8ec40b5e", "0339e7901bccba1e3c8e05956536823b2b0e7189c66f5796b7602b63a8fd1ff9", "b213bb94b274991d4288a6405954059e99b4c4b891a74a1abcd83ea295331b18", "d0a7195ec0cd987709b4dc6416e0ed6fc9939054ecbf502da8c4c6a09836ed9c", "7b9c334b3aeb75a795f9d6c7c0ee01ab219f31860880eb3480921dcd2a057d2b", "9c4e722d126467603530d242fe91a19fa90ebd3a461ee38f36ef5eefa07e996c", "4306ac8ccd2ce6a880350f95c7d59635371ba3d78bb13353c5b7ff06f7c6fc40", "4b9360e2d86f20850d2c6ff222ed16c6a4252c00afad8d488c30c162b3a10da7", "927f20b9dcfbb80f4a6b5d6067a586835bdcb5f3e921ed87bec67fb5160181d1", "e620bc51fbeb8011f57324b0a7ae6f45c46050cd624887f0a50879880632fdaa", "ee7b749b81e86d46fa3e93b9aba29285bae38a91f175dbf7c619d05fcf91e857", "573d5039fa570ceb3fa136be73c432b49a19af00a7f109325b78160f7dc13db1", "9ab1936825e830d4eab7a945701528579f78a8d1702a76a774e7456ddd3a254e", "2b3538a6fed897c0143f51b82f7e9e1929cb698e7de8d88aa8b1d23cabd58fa9", "21e2f8ae0522da985262ccf8422d98d75068ccd448d15c4bfec9f793713c7644", "c02a276e24fbb64f5b35d4b6555d1d873095e076868cea8dcfdad9e606612f9b", "7756adb6b470c5126693a4de57c1d5b38afab4f7ffc4f982374e8466051bcfca", "f82cbe9343e63fa4bf486f8e4113f91abef7c994e6f7068b500942fede79f095", "782f9df4e3f669149a575922a7318d523b1ab8a5911a2b1c2850839d5762cf03", "89ef33e05604e28f762b3cdf2f20d876adcb104a87c2636c5facb61ec47d020c", "59e374462a0c7e32df5e087d4d250936ef54aa19ca824ebaa63b66406180719d", "11fc2b68e458f12e93398a453c5efac599691bd89d40c35e003dc594d87bf51d", "5f793edc159efab968da834bd44187fff951cec822ca1b8982b1f36d966956be", "da0d474d5e0ec5d0966e1986a5de3f085e0f491da67cdb43d52fdc9848b14314", "8d4eec56231819d18f3fb3ec6e6881b269c0ccb881eedecb5916d2b4ef82c6cf", "137e7ea7c47a724f8a4494a3e73e74f146282382935d64d25385dd720f537e98", "1a2a9c7707443c848897141a4f659fbd0b7fefa47365f2af43183777dcb4a8ef", "7747a6f738959e6d75f16fe6d0782b455258b9c93d0380a230722cd6ae11e0bb", "314e30caef6c7c09b2a85056610949febb6abbbf7702c5d6706cef658123d782", "9ab42848b175c62790b5aa4f256899bb609d05723d364b8d349160afadfd9f95", "853b07dda09eb155dcebbac23e2fa5d76c5f619f3cabfa5e25fd82706485bd25", "a2b0053632aafe21d4dff287c03c362cae2a1d3267cd87d82a7ba9a3795129c9", "7918541145cb2c5918b8fa20a31298a7bc9b8f43aebb69f046f78d070a7f22ef", "0827e91cf9ec4dbb95966d68cdeb90dc8399457f47922d1e53eb2972c87756ef", "6121dac0131fc1fe0f7652d6c2195141c0e6a9b7e5cb555647ec3bb2f90b912d", "134fae4eec772042a832efc19e2f3e449db962f3573c070f2920591c306967b1", "b9a716636f3d1dd47e61aa1216f55317230cf734e06c9f740552f2bbd6e8210a", "d5caa5c0bb57e75c78de5f6f132e19776b777dd205d37ff6c2179412caa32c40", "e11c15139b71e7078a664d430e115c631ac8cdd89a8f4b35e4bbbeb9ec85dc17", "cbff909b284e4b1858adff2a0cee75032a2b2411d805604dfe820e40e855d6b5", "5b4ce1b89dde6b8b5cbec1b454306b7f53a9dadcdbe5df429ea5a33635d989d3", "c06a55411e962e0bf9cc11c14e854be084906b374cc181868c29ebcab0b66775", "ad16c4f73055baa8c0c6f69e294019ea90e3e97ee90923c4478156e15180d19c", "76866d7b50747a469e9891c529b7a58a4b9082d113b7acbe2b46f6049a8d36c7", "df96c9eba4763a1c3a8a0d2eb14e57847ce679adeda80b04cb86ef4f40cf290a", "6421d33aed4529b00db819051abed4ae78f28778feab921177c24378d48b427b", "cb76cbf3c146f5890eef6a8e78349b9291b75d2ca3b947b027f52dab0acbcdd4", "cb9a9e1606d5d6cc59bce096733be7e6902d8c8de19d22cc0f5435ad4e719015", "d3b9005c6b93a657d8edd2312d4d59b8807ea7c509079dfd1e4a8cef3d6852ba", "ebc705fd3ee20a69c5e99b1bf063acff8c926eec9358a36294b8df0fdcd31eb5", "c99e64329e066cc19b2e9962bfa2eb474bb7f9bd1c797421878209c16ca85d80", "55a081aab8afb0cfe83873b812c4495a762bdfc866d74c038d64f73d26944db3", "d830b389b67743e2a2cec5d64af37ce1b991b2781bf2a3fb1e8283bc78e98495", "558d06ff221f4d6e5265465ef2928828a80b498f95d7b1853c4a93d842931ccd", "e967f7ac0177971566b44535eed88a5ffcd0b2ec09de03edbf817f8e110eaf5b", "404df2a8bcf278cae68d9a43b86ff9c2781461ccd227c20aa5e0c5b1db2c0cb1", "f8f5160a6d1e91a3cae676b1e8f8563da2e1cb92869df51c190f0d91f62c81b2", "30a23be3cb0e3feab447217745d537e6c5299f3a95172c234bb84de54169b694", "7c5e66106c5e7cb9e68cd6bec431acdb4b0c9394f2c000a60f0ec558b1667750", "be103be330df170331a747138325af15173704afb808abfd6fc5742c677de241", "d711f0d3914c1bee36324e055ade9058750f2b3d0206f516382702de8eda3757", "519658c8746832821044b074a40661ea1497ca50426888303d8eec43ae8b9d6a", "87cd56d2f6ff774a0c75b029c2a888df7b41319380336f3e4663fe5417229687", "2efe240e7018fd0443262223d286c04120199063f4ef194bdef9af0ab34fa4a8", "8c9a69c950bea4e4beecc286124bf44e2cc78614f767580d59dc22cf94bd23f6", "e7641851ddf32f8fa1937528a2c88a2ef512d45f0a7296c232df6584471ad7ba", "a5beb770e26085eb45a6c5e15acb5844fdda167261e92b20c87dc72c1e0d0a1d", "f54988150d2ba3327251b7a4672ec9bff6fe93f06a7a9f19030f17e693281f11", "6cbfa48ae32ef9b3798f0afe4b86798497a758735dc3ac3e0aa6b42710476f58", "35130215ec7db0e57d5964dacb9aa2ea858e70fc864edd08cf062334823a3ce8", "a935e9ddece310c12baa815a0077e151b300a293f88651d7715ea33151d4016e", "167e10bb4d35aa27a4916de2f846ff5d323a0090c9d37b9c35ca455272ab07be", "a85a1222927f535ca37587d38ab4db2bc940bfc0c6d703003119329d05469a75", "826ab7e279754c009dcd86421f3bdaaa3325bdfff8352788c9f8cfbdddfcfafe", "8336015f3f6ca5d69d5af6dfd521a3e3c024c08121bd42de3a25e5bffb417d42", "194125cbe3f428afbf59da1dd144062ad288011e10beca10ca534f935ea7290d", "3bb36e7a0165d3b6f51b628c18e6b4d9e355b05c5be7a616c881dc395c623c66", "c092f7add11cee0facec22c78badba46fb8688538df1443b7356ceea83bae10d", "31552a2bb308a5778e815fee39b007ce5a633d2e7ba27f08eee2bec6f8d387b7", "373533933e0aae2d2dcffb59b09c49fa64506606aa0359eddf00326ee7bbcc7e", "0f580299cabe89b2dc9809735d14fdabf60cc1b65824bb5f6b5cf283b68210ee", "138c02ce7b36a4d7e82f942a3291bbb357b2e8845b579189ce4c35e01e6b859f", "9dc184037f271c4043b1a6d01d9fbed5d2f156fb561ec2612e5b1cd6aa486083", "cb2ae942cd73059bfe666d9ef78cee5a557cda842c9503df0f7d6b00be815cc5", "f941433597eaa923318023f040798918f743db7bf6d33bc6a13bc8c2e8d3e711", "02a1f2c523e2705b1ab122a06c08bd64080ef76d09d517c56c4e64a3f6626021", "ac3dda90e10c66d26ebb6911924713785f48e8e3d2150aa06ae90db456e1c9a0", "61a39a58e915f953d1ea5c0483f3f45b33ed6f097d76ea6d03d7cf81616f33bd", "b3ff677201fa7543da2f635753305a128c4076409268f1ee53ee824989193e90", "3a2cf44822616731ce40cde80365738e4a4d9af161de3cc2bb3e4f4d3ced8009", "b1a3f23c441a6afece152c4b2e1f1da6fc952f997bc8711a6122e26afafeb5b1", "87123bd9968d64fead15b346ad4ef3b0918aebc596fb7ce8c016c09085985bbd", "eae98597fc685154c882a62073157e1538e37270573de17e7f9bd1af724e1164", "e6dc4cfe6c4b77ebc2a915a49157447a65f85c275ba6c888fddbfae95a2d1c2b", "45ffffa2166eff3624a6b83e5d953669e3639188556330a58656d51ac9008f15", "2b5658b7d00f6d34890e71cf1d57b520e934f6b4087cca5c50604a7c8190488d", "d9b516ec359cccafc8cd2c5721bed137cb0d4b7bb21ba4772baed786a9f059a6", "5005e282fff3675ff3ac18906d5cf9df5b992d0bd95fc9cd3258f386f1c5b5ea", "2dea763455c4ae2c662bd9db6529b85cfd397744cb3da1a639925b0fa2b048b4", "497399dc295066a487984ab67cbfec9bf3d65184bc424a7b96268f2c03e6557f", "8f87e5ab712b41e1bc6f74fd74bb8e96323f62f62bedb35ed578992ddbbd5f47", "a5504fbce2afcd7277b0bd94581050195607d5c6701cff8d8e25f05a2d50d81c", "205b534ee10a3633f87c8ab36590d114f516985470ef5851077ac5c95aa83f16", "0d2093c088c08840643f542a44d9e8c389694f03dc9c62a264445de5758e73c3", "b32c1de573b72b62ce6b77d628f758acbfe89ecaa17d3c4c94cad8dff45dd0c9", "6d75d744de2e5dd7ebd3fa47b22ca0d99d4255ee36b5e767567479e0134e0697", "d3228e2e8e5de7178f2afc4b6f86b13287469b55410a164397bc602a0e3bd2db", "5d0a5e9e280f90c7d1f69b69ff3b5bfb94bce299dde8799520fe92912afd2cff", "aafabddd3fe15559af9138aa113c2473fed25a41ee52877a05dc2f9b24416827", "00a9160b3ae08d4066e53992f3cc004b3f6bf3d840613d6e847fb16323ddb270", "1473078fe8d18a5e3f791064c1083783fdc19517a3f2af47777d8778bb2b2f89", "aa2b720f1b7fd016086641fa0c3a6f8133c5f7eb3e9a65cd01ad0b51e7c35719", "ecdd45371e9a284e97416f414d665afa0aec864277a03c333e785e4d6ba6d439", "66a7301e8f3d54360b15fc64610398888301a3caeb685dc71e0ec0fdd175937f", "bd156dc25f23d82eaef927957d4c8c883ec0c80de4c58310313764ccc701d281", "3e8aa53535920d5886779d30687c2350800e9c712c5c2414db463b9c99f3052e", "308a237dad23fa158e7590ce7c75e788ec3ae6be8f6972a867f2eb94f6417c96", "4b12d020e1df286f672fe5d2eac74d95f817d0bbb8bee15a7913ebd9c3a8014a", "303c6f66eaff75bf2145e3bcc343245bcbedb2df46af1fb1e8382473fd2ab402", "d21e974892bd9209a0e2333b22acb55ec2a4abc015755379640cb81d4ba38d82", "40bdb0c10ce735f5e6abf18bf46dd8ef5625ea828fbfc6e380b70809d7cf76dd", "c0b4d28f557f71bcc41eb3573e2afb6da0c127639972bbcb8f4962cff0896f7a", "d2e36f3773f4c313fafb160ac753f1a11b53783920d45552b693f7a37b80bbe2", "3fd160ad0045137801256a22fed09f5f31aacf31f1681fbf6d70bc03972d2253", "2c0c05796774bdbb27c0a6ec5559817b4cd48feee80dff2c540257f86733e397", "5f17ad7ebf06c9ee5f7c86716e2392fd65b773eb6c94f47ac1ea1e12afbacfd0", "0dc16b207a0a9a722cd0b6ce18419eeb2c7809a9f90f3ebca7cc084d6714469d", "8f576a107b37c1309055282825effed4d57dd7e96fd69595ad300c26f77b07a5", "b433f6a339e84a5dbd8e6638a4547dd029b642d1007199948678d7574350b64e", "738768e552067738d3ba97fabe8ea93c0a6ba3b64cc24fab0e9b0c2ce4842982", "533020acb857afd489d4766280665cc484d184ed8eeaacd031e8a5e70b5c4a88", "1d84007a810cb751a5f7207b36cffb1a7f50c1553cbcb0c922c7cb1ada8bb409", "a0b398eb392174cfa24948edcf03c50553a7367c7f6ed50970456484ea09680b", "f156f642f5fd502eb9d0fff911981506c32e6c40b12362e6b3082dffb7fc6550", "2338e90aafd734d44bd50aad3f4d0f4255e2d2505546925e810798626c79f4f4", "c141f87ed878c297468d5be367ce8df0c7d90be4b6be070059eb9345f8250b62", "57106030bb89bd435844ef9baf318c9696af10784a4cf09359bff4b22a4d74eb", "2419aa33614ded3307173c53d6f614b6567d6f50fdc9a99fd32a299efc3de982", "07e60b9438f0b0fc97151c34b781b2a6370cb4d6c48ecbbfe0016a24ebe7bf31", "c09518a1b22c36e3d599af9f956090609fff015a794680a12f730364c721aae4", "65ddb5cf2927237525c5b3d3613eb346660cba60d0478ea917b6f0aa4907d7d9", "1c8935ede01448904447520b90c742615062e404f3525fe5bd667e06f7341c13", "fcb9e121eb526413ec8c827a3dda5e619a85ffbcb7508f0525ac22a121a100f5", "6d0a6422309f64d722ba79f621a4fe3db0ecf16b40366313b146a97d95667307", "4ad7e9d2a199b2eb3cc1cf7bb35e7b03a0ca18bd7382ed29a18b97ed01cd63a0", "69e377941f0263ce3c585789ae6106782d1f15db0b1942a9627b2bd6fe83e13d", "bb51ed5948d59b0dcb2f5cb5f8a27d3f70b8c71660b0d6d4ab658b6a7ca2356c", "6695e79e0e07fde8c05da60736ba373d55271d5a7c6da2a2c7d30e957a46e7e7", "48bd888c98b158b5c82b148f091a91bb1881b9a1931227f0a5269649a8eebaff", "771382cfa5138ccd32fdddad18e3eb8f1a06eee10704248d1e4d49f32872afe6", "176bb2e118aeb292912fa1903470621ae385e819a50c580301b33165666f3c7d", "15596f8c5f8fb397e5214e6f5eaf286a813b6e5d8bebae2bad1d550511f92840", "bdacbb2d763783f1ac51fd2477276543f79db13a434697a2aedd8523a1427e1f", "ca0e3b746890e8d626840d445989bb0e703f3e4c792aaa49a6b8952ea7696063", "1319af4c3801a463f0e1b7a9cfe2cfbb79e769fb0daed1a2868ade7665765ea6", "172f67582c5270cf0ef8264ef64bc5e17a53aac87693eff1860dfe56aea4209e", "0462589f719e853654d1ca00038dfc806ae7acb9bb5a3f9e6d458f3d4206f532", "f7480a6f46b553517f41238cbd5a6069eab164fd1512e1685f9bddf5c1afa59c", "a5cfdbe5c0b38b0904b5fe6afc2ce583dce1dbc7b4cd88224cbd88efa30b0291", "6f07b548ced6405ef78693332d516d041780f85f0771cfbaba8bbb86a6cdfb7d", "de0184abac150e780e26f1e7de09da64dfee433e8c9a9efe8d93a673350016b8", "8e7cee539c6315ad939a9495e40e7e70e2d07f6b2920cdbcc689457cd9e11997", "0088ccc025bf814e8098607bfbd17448024495a62610700b6000ec448afc1ca3", "d3a0503fdb8802e979871dca7d3c10a928cedf1978e44f42ecb72b96ada13dc3", "add0b405d079dd0c682a1e5026ef1a5b989b0bdf044d2db28249b4d51a74c5dc"}

	// Convert each string to a [32]byte
	for i := 0; i < 256; i++ {
		// Decode hex string to bytes
		mainnetFixedHashes[i] = gethCommon.HexToHash(mainnetHashes[i])
		testnetFixedHashes[i] = gethCommon.HexToHash(testnetHashes[i])
	}

	fixedHashes = make(map[flow.ChainID][256]gethCommon.Hash)
	fixedHashes[flow.Mainnet] = mainnetFixedHashes
	fixedHashes[flow.Testnet] = testnetFixedHashes
}

func init() {
	generateFixedHashes()
}
