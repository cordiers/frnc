package chain

func Init(id, dataDir string, validatorConfigs []*ValidatorConfig) (*Chain, error) {
	chain, err := new(id, dataDir)
	if err != nil {
		return nil, err
	}
	if err := initNodes(chain, len(validatorConfigs)); err != nil {
		return nil, err
	}
	if err := initGenesis(chain); err != nil {
		return nil, err
	}
	if err := initValidatorConfigs(chain, validatorConfigs); err != nil {
		return nil, err
	}
	return chain.export(), nil
}
