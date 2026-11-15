package cultivation

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxNameLength        = 120
	maxSpeciesLength     = 120
	maxVarietyLength     = 120
	maxCategoryLength    = 80
	maxLocationLength    = 200
	maxSubstrateLength   = 200
	maxContainerLength   = 120
	maxTechniqueLength   = 200
	maxOutcomeLength     = 2000
	maxEntryBodyLength   = 5000
	maxMeasurementKeys   = 20
	maxMeasurementKey    = 40
	maxMeasurementValue  = 80
	maxListItemLength    = 120
	maxListItems         = 30
	maxHarvestUnitLength = 20
)

func trim(value string) string {
	return strings.TrimSpace(value)
}

// NormalizeCycleInput remove espaços e aplica valores padrão.
func NormalizeCycleInput(input CycleInput) CycleInput {
	input.Name = trim(input.Name)
	input.Species = trim(input.Species)
	input.Variety = trim(input.Variety)
	input.Category = trim(input.Category)
	input.Location = trim(input.Location)
	input.Substrate = trim(input.Substrate)
	input.Container = trim(input.Container)
	input.Technique = trim(input.Technique)
	if input.Environment == "" {
		input.Environment = EnvironmentOutdoor
	}
	if input.Stage == "" {
		input.Stage = StageGermination
	}
	if input.Visibility == "" {
		input.Visibility = VisibilityPrivate
	}
	return input
}

// ValidateCycleInput valida os dados de criação de um ciclo.
func ValidateCycleInput(input CycleInput) error {
	if err := validateCycleCommon(input.Name, input.Species, input.Variety, input.Category, input.Environment, input.Location, input.Substrate, input.Container, input.Technique, input.Stage, input.Visibility); err != nil {
		return err
	}
	if input.StartedOn.IsZero() {
		return invalid("started_on", "Informe a data de início do cultivo.")
	}
	return nil
}

// NormalizeCycleUpdateInput remove espaços e aplica valores padrão.
func NormalizeCycleUpdateInput(input CycleUpdateInput) CycleUpdateInput {
	input.Name = trim(input.Name)
	input.Species = trim(input.Species)
	input.Variety = trim(input.Variety)
	input.Category = trim(input.Category)
	input.Location = trim(input.Location)
	input.Substrate = trim(input.Substrate)
	input.Container = trim(input.Container)
	input.Technique = trim(input.Technique)
	input.Outcome = trim(input.Outcome)
	if input.Environment == "" {
		input.Environment = EnvironmentOutdoor
	}
	if input.Stage == "" {
		input.Stage = StageGermination
	}
	if input.Status == "" {
		input.Status = StatusActive
	}
	if input.Visibility == "" {
		input.Visibility = VisibilityPrivate
	}
	return input
}

// ValidateCycleUpdateInput valida os dados de edição de um ciclo.
func ValidateCycleUpdateInput(input CycleUpdateInput) error {
	if err := validateCycleCommon(input.Name, input.Species, input.Variety, input.Category, input.Environment, input.Location, input.Substrate, input.Container, input.Technique, input.Stage, input.Visibility); err != nil {
		return err
	}
	if !input.Status.Valid() {
		return invalid("status", "Situação do ciclo inválida.")
	}
	if input.StartedOn.IsZero() {
		return invalid("started_on", "Informe a data de início do cultivo.")
	}
	if utf8.RuneCountInString(input.Outcome) > maxOutcomeLength {
		return invalid("outcome", "O resultado deve ter no máximo 2000 caracteres.")
	}
	if input.FinishedOn != nil && input.FinishedOn.Before(input.StartedOn) {
		return invalid("finished_on", "A data de encerramento não pode ser anterior à data de início.")
	}
	return nil
}

func validateCycleCommon(name, species, variety, category string, environment Environment, location, substrate, container, technique string, stage Stage, visibility Visibility) error {
	if length := utf8.RuneCountInString(name); length == 0 || length > maxNameLength {
		return invalid("name", "Informe um nome de ciclo com até 120 caracteres.")
	}
	if length := utf8.RuneCountInString(species); length == 0 || length > maxSpeciesLength {
		return invalid("species", "Informe a espécie com até 120 caracteres.")
	}
	if utf8.RuneCountInString(variety) > maxVarietyLength {
		return invalid("variety", "A variedade deve ter no máximo 120 caracteres.")
	}
	if utf8.RuneCountInString(category) > maxCategoryLength {
		return invalid("category", "A categoria deve ter no máximo 80 caracteres.")
	}
	if !environment.Valid() {
		return invalid("environment", "Ambiente inválido.")
	}
	if utf8.RuneCountInString(location) > maxLocationLength {
		return invalid("location", "A localização deve ter no máximo 200 caracteres.")
	}
	if utf8.RuneCountInString(substrate) > maxSubstrateLength {
		return invalid("substrate", "O substrato deve ter no máximo 200 caracteres.")
	}
	if utf8.RuneCountInString(container) > maxContainerLength {
		return invalid("container", "O recipiente deve ter no máximo 120 caracteres.")
	}
	if utf8.RuneCountInString(technique) > maxTechniqueLength {
		return invalid("technique", "A técnica deve ter no máximo 200 caracteres.")
	}
	if !stage.Valid() {
		return invalid("stage", "Estágio inválido.")
	}
	if !visibility.Valid() {
		return invalid("visibility", "Visibilidade inválida.")
	}
	return nil
}

// NormalizeEntryInput remove espaços e aplica valores padrão.
func NormalizeEntryInput(input EntryInput) EntryInput {
	input.Body = trim(input.Body)
	input.HarvestUnit = trim(input.HarvestUnit)
	input.PracticeID = trim(input.PracticeID)
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now()
	}
	if input.Visibility == "" {
		input.Visibility = EntryVisibilityInherit
	}
	if input.Measurements == nil {
		input.Measurements = map[string]string{}
	}
	input.Interventions = normalizeList(input.Interventions)
	input.Occurrences = normalizeList(input.Occurrences)
	return input
}

// ValidateEntryInput valida os dados de um registro de cultivo.
func ValidateEntryInput(input EntryInput) error {
	if input.OccurredAt.IsZero() {
		return invalid("occurred_at", "Informe a data e a hora do registro.")
	}
	if utf8.RuneCountInString(input.Body) > maxEntryBodyLength {
		return invalid("body", "O texto do registro deve ter no máximo 5000 caracteres.")
	}
	if input.Stage != nil && !input.Stage.Valid() {
		return invalid("stage", "Estágio inválido.")
	}
	if !input.Visibility.Valid() {
		return invalid("visibility", "Visibilidade inválida.")
	}
	if len(input.Measurements) > maxMeasurementKeys {
		return invalid("measurements", "Informe no máximo 20 medidas.")
	}
	for key, value := range input.Measurements {
		if length := utf8.RuneCountInString(key); length == 0 || length > maxMeasurementKey {
			return invalid("measurements", "Cada medida deve ter um nome de até 40 caracteres.")
		}
		if utf8.RuneCountInString(value) > maxMeasurementValue {
			return invalid("measurements", "Cada valor de medida deve ter no máximo 80 caracteres.")
		}
	}
	if len(input.Interventions) > maxListItems {
		return invalid("interventions", "Informe no máximo 30 intervenções.")
	}
	for _, item := range input.Interventions {
		if length := utf8.RuneCountInString(item); length == 0 || length > maxListItemLength {
			return invalid("interventions", "Cada intervenção deve ter até 120 caracteres.")
		}
	}
	if len(input.Occurrences) > maxListItems {
		return invalid("occurrences", "Informe no máximo 30 ocorrências.")
	}
	for _, item := range input.Occurrences {
		if length := utf8.RuneCountInString(item); length == 0 || length > maxListItemLength {
			return invalid("occurrences", "Cada ocorrência deve ter até 120 caracteres.")
		}
	}
	if input.HarvestQuantity != nil && *input.HarvestQuantity < 0 {
		return invalid("harvest_quantity", "A quantidade colhida não pode ser negativa.")
	}
	if utf8.RuneCountInString(input.HarvestUnit) > maxHarvestUnitLength {
		return invalid("harvest_unit", "A unidade deve ter no máximo 20 caracteres.")
	}
	return nil
}

func normalizeList(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		item = trim(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}
