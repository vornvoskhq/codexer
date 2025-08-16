package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"plandex-server/db"

	shared "plandex-shared"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

const CustomModelsMinClientVersion = "2.2.0"

func UpsertCustomModelsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateCustomModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	var modelsInput shared.ModelsInput
	if err := json.NewDecoder(r.Body).Decode(&modelsInput); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(modelsInput.CustomProviders) > 0 {
		if os.Getenv("IS_CLOUD") != "" {
			http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
			return
		}
	}

	if len(modelsInput.CustomModels) > 0 {
		if os.Getenv("IS_CLOUD") != "" {
			apiOrg, err := getApiOrg(auth.OrgId)
			if err != nil {
				log.Printf("Error fetching org: %v\n", err)
				http.Error(w, "Failed to create custom model: "+err.Error(), http.StatusInternalServerError)
				return
			}

			if apiOrg.IntegratedModelsMode {
				http.Error(w, "Custom models are not supported on Plandex Cloud in Integrated Models mode", http.StatusBadRequest)
				return
			}
		}
	}

	hasDuplicates, errMsg := modelsInput.CheckNoDuplicates()
	if !hasDuplicates {
		http.Error(w, "Has duplicates: "+errMsg, http.StatusBadRequest)
		return
	}

	for _, provider := range modelsInput.CustomProviders {
		if provider.Name == "" {
			msg := "Provider name is required"
			log.Println(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}

	for _, model := range modelsInput.CustomModels {
		if model.ModelId == "" {
			msg := "Model id is required"
			log.Println(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		if shared.BuiltInBaseModelsById[model.ModelId] != nil {
			msg := fmt.Sprintf("%s is a built-in base model id, so it can't be used for a custom model", model.ModelId)
			log.Println(msg)
			http.Error(w, msg, http.StatusUnprocessableEntity)
			return
		}
	}

	for _, modelPack := range modelsInput.CustomModelPacks {
		if modelPack.Name == "" {
			msg := "Model pack name is required"
			log.Println(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		if shared.BuiltInModelPacksByName[modelPack.Name] != nil {
			msg := fmt.Sprintf("%s is a built-in model pack name, so it can't be used for a custom model pack", modelPack.Name)
			log.Println(msg)
			http.Error(w, msg, http.StatusUnprocessableEntity)
			return
		}
	}

	var existingCustomModelIds = make(map[shared.ModelId]bool)
	var existingCustomProviderNames = make(map[string]bool)

	customModels, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to create custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	customModelPacks, err := db.ListModelPacks(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom model packs: %v\n", err)
		http.Error(w, "Failed to create custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var customProviders []*db.CustomProvider

	if os.Getenv("IS_CLOUD") == "" {
		customProviders, err = db.ListCustomProviders(auth.OrgId)
		if err != nil {
			log.Printf("Error fetching custom providers: %v\n", err)
			http.Error(w, "Failed to create custom model: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	apiCustomModels := make([]*shared.CustomModel, len(customModels))
	for i, model := range customModels {
		apiCustomModels[i] = model.ToApi()
	}

	apiCustomProviders := make([]*shared.CustomProvider, len(customProviders))
	for i, provider := range customProviders {
		apiCustomProviders[i] = provider.ToApi()
	}

	apiCustomModelPacks := make([]*shared.ModelPackSchema, len(customModelPacks))
	for i, modelPack := range customModelPacks {
		apiCustomModelPacks[i] = modelPack.ToApi().ToModelPackSchema()
	}

	updatedModelsInput := modelsInput.FilterUnchanged(&shared.ModelsInput{
		CustomModels:     apiCustomModels,
		CustomProviders:  apiCustomProviders,
		CustomModelPacks: apiCustomModelPacks,
	})

	for _, model := range customModels {
		existingCustomModelIds[model.ModelId] = true
	}

	for _, provider := range customProviders {
		existingCustomProviderNames[provider.Name] = true
	}

	inputModelIds := make(map[string]bool)
	inputProviderNames := make(map[string]bool)
	inputModelPackNames := make(map[string]bool)

	for _, model := range modelsInput.CustomModels {
		inputModelIds[string(model.ModelId)] = true
	}

	for _, provider := range modelsInput.CustomProviders {
		inputProviderNames[provider.Name] = true
	}

	for _, modelPack := range modelsInput.CustomModelPacks {
		inputModelPackNames[modelPack.Name] = true
	}

	var toUpsertCustomModels []*db.CustomModel
	var toUpsertCustomProviders []*db.CustomProvider
	var toUpsertModelPacks []*db.ModelPack

	for _, provider := range updatedModelsInput.CustomProviders {
		dbProvider := db.CustomProviderFromApi(provider)
		dbProvider.Id = provider.Id
		dbProvider.OrgId = auth.OrgId

		toUpsertCustomProviders = append(toUpsertCustomProviders, dbProvider)
	}

	for _, model := range updatedModelsInput.CustomModels {
		// ensure that providers to upsert are either built-in, being imported, or already exist
		for _, provider := range model.Providers {
			if provider.Provider == shared.ModelProviderCustom {
				_, exists := existingCustomProviderNames[*provider.CustomProvider]
				_, creating := inputProviderNames[*provider.CustomProvider]
				if !exists && !creating {
					msg := fmt.Sprintf("'%s' is not a custom model provider that exists or is being imported", *provider.CustomProvider)
					log.Println(msg)
					http.Error(w, msg, http.StatusUnprocessableEntity)
					return
				}
			} else {
				pc, builtIn := shared.BuiltInModelProviderConfigs[provider.Provider]
				if !builtIn {
					msg := fmt.Sprintf("'%s' is not a built-in model provider", provider.Provider)
					log.Println(msg)
					http.Error(w, msg, http.StatusUnprocessableEntity)
					return
				}
				if os.Getenv("IS_CLOUD") != "" && pc.LocalOnly {
					msg := fmt.Sprintf("'%s' is a local-only model provider, so it can't be used on Plandex Cloud", provider.Provider)
					log.Println(msg)
					http.Error(w, msg, http.StatusUnprocessableEntity)
					return
				}
			}
		}

		dbModel := db.CustomModelFromApi(model)
		dbModel.Id = model.Id
		dbModel.OrgId = auth.OrgId

		toUpsertCustomModels = append(toUpsertCustomModels, dbModel)
	}

	for _, modelPack := range updatedModelsInput.CustomModelPacks {
		// ensure that all models are either built-in, being imported, or already exist
		allModelIds := modelPack.AllModelIds()

		for _, modelId := range allModelIds {
			_, exists := existingCustomModelIds[modelId]
			_, creating := inputModelIds[string(modelId)]
			bm, builtIn := shared.BuiltInBaseModelsById[modelId]

			if !exists && !creating && !builtIn {
				msg := fmt.Sprintf("'%s' is not built-in, not being imported, and not an existing custom model", modelId)
				log.Println(msg)
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}

			if builtIn && os.Getenv("IS_CLOUD") != "" && bm.IsLocalOnly() {
				msg := fmt.Sprintf("'%s' is a local-only built-in model, so it can't be used on Plandex Cloud", modelId)
				log.Println(msg)
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}
		}

		mp := modelPack.ToModelPack()
		dbMp := db.ModelPackFromApi(&mp)
		dbMp.OrgId = auth.OrgId
		dbMp.Id = mp.Id

		toUpsertModelPacks = append(toUpsertModelPacks, dbMp)
	}

	toDeleteCustomModelIds := []string{}
	toDeleteCustomProviderIds := []string{}
	toDeleteModelPackIds := []string{}

	for _, model := range customModels {
		if _, exists := inputModelIds[string(model.ModelId)]; !exists {
			toDeleteCustomModelIds = append(toDeleteCustomModelIds, model.Id)
		}
	}

	for _, provider := range customProviders {
		if _, exists := inputProviderNames[provider.Name]; !exists {
			toDeleteCustomProviderIds = append(toDeleteCustomProviderIds, provider.Id)
		}
	}

	for _, modelPack := range customModelPacks {
		if _, exists := inputModelPackNames[modelPack.Name]; !exists {
			toDeleteModelPackIds = append(toDeleteModelPackIds, modelPack.Id)
		}
	}

	numChanges := len(toUpsertCustomModels) + len(toUpsertCustomProviders) + len(toUpsertModelPacks) + len(toDeleteCustomModelIds) + len(toDeleteCustomProviderIds) + len(toDeleteModelPackIds)
	if numChanges == 0 {
		w.WriteHeader(http.StatusOK)
		log.Println("No changes to custom models/providers/model packs")
		return
	}

	err = db.WithTx(r.Context(), "create custom models/providers/model packs", func(tx *sqlx.Tx) error {
		for _, model := range toUpsertCustomModels {
			if err := db.UpsertCustomModel(tx, model); err != nil {
				return fmt.Errorf("error creating custom model: %w", err)
			}
		}

		for _, provider := range toUpsertCustomProviders {
			if err := db.UpsertCustomProvider(tx, provider); err != nil {
				return fmt.Errorf("error creating custom provider: %w", err)
			}
		}

		for _, modelPack := range toUpsertModelPacks {
			if err := db.UpsertModelPack(tx, modelPack); err != nil {
				return fmt.Errorf("error creating model pack: %w", err)
			}
		}

		if len(toDeleteCustomModelIds) > 0 {
			if err := db.DeleteCustomModels(tx, auth.OrgId, toDeleteCustomModelIds); err != nil {
				return fmt.Errorf("error deleting custom models: %w", err)
			}
		}

		if len(toDeleteCustomProviderIds) > 0 {
			if err := db.DeleteCustomProviders(tx, auth.OrgId, toDeleteCustomProviderIds); err != nil {
				return fmt.Errorf("error deleting custom providers: %w", err)
			}
		}

		if len(toDeleteModelPackIds) > 0 {
			if err := db.DeleteModelPacks(tx, auth.OrgId, toDeleteModelPackIds); err != nil {
				return fmt.Errorf("error deleting model packs: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		http.Error(w, "Failed to import custom models/providers/model packs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully imported custom models/providers/model packs")
}

func GetCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetCustomModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	id := mux.Vars(r)["modelId"]

	res, err := db.GetCustomModel(auth.OrgId, id)
	if err != nil {
		log.Printf("Error fetching custom model: %v\n", err)
		http.Error(w, "Failed to fetch custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if res == nil {
		http.Error(w, "Custom model not found", http.StatusNotFound)
		return
	}

	err = json.NewEncoder(w).Encode(res.ToApi())
	if err != nil {
		log.Printf("Error encoding custom model: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom model: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom model")
}

func ListCustomModelsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListCustomModelsHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	models, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to fetch custom models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiList []*shared.CustomModel
	for _, m := range models {
		apiList = append(apiList, m.ToApi())
	}

	err = json.NewEncoder(w).Encode(apiList)
	if err != nil {
		log.Printf("Error encoding custom models: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom models: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom models")
}

func GetCustomProviderHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	id := mux.Vars(r)["providerId"]

	res, err := db.GetCustomProvider(auth.OrgId, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(res.ToApi())
	if err != nil {
		log.Printf("Error encoding custom provider: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom provider: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom provider")
}

func ListCustomProvidersHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	list, err := db.ListCustomProviders(auth.OrgId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apiList []*shared.CustomProvider
	for _, p := range list {
		apiList = append(apiList, p.ToApi())
	}

	err = json.NewEncoder(w).Encode(apiList)
	if err != nil {
		log.Printf("Error encoding custom providers: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom providers: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom providers")
}

func CreateModelPackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateModelPackHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	http.Error(w, "Use POST /custom_models instead to create model packs", http.StatusBadRequest)
}

func UpdateModelPackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateModelPackHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	http.Error(w, "Use POST /custom_models instead to update model packs", http.StatusBadRequest)
}

func ListModelPacksHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListModelPacksHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	sets, err := db.ListModelPacks(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching model packs: %v\n", err)
		http.Error(w, "Failed to fetch model packs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiPacks []*shared.ModelPack

	for _, mp := range sets {
		apiPacks = append(apiPacks, mp.ToApi())
	}

	json.NewEncoder(w).Encode(apiPacks)

	log.Println("Successfully fetched model packs")
}
