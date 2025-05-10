#\!/bin/bash
sed -i -e 's/NewLLMFactory/NewFactory/g' -e 's/client, err := factory.GetClient("mock")/client, err := factory.GetClient(providers.ProviderMock)/g' /Users/lancerogers/Dev/AI/Guild/cmd/guild/objective_ui.go
