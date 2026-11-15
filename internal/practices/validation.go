package practices

import (
	"strings"
	"unicode/utf8"
)

const (
	maxTitleLength       = 160
	maxSummaryLength     = 500
	maxCategoryLength    = 80
	maxEstimateLength    = 80
	maxMaterials         = 50
	maxMaterialLength    = 200
	maxSteps             = 50
	maxStepLength        = 1000
	maxSafetyLength      = 2000
	maxAdaptationsLength = 2000
	maxReferencesLength  = 2000
	maxLicenseLength     = 120
)

// NormalizeInput remove espaços, descarta linhas vazias e aplica padrões.
func NormalizeInput(input Input) Input {
	input.Title = strings.TrimSpace(input.Title)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Category = strings.TrimSpace(input.Category)
	input.TimeEstimate = strings.TrimSpace(input.TimeEstimate)
	input.CostEstimate = strings.TrimSpace(input.CostEstimate)
	input.Safety = strings.TrimSpace(input.Safety)
	input.Adaptations = strings.TrimSpace(input.Adaptations)
	input.References = strings.TrimSpace(input.References)
	input.License = strings.TrimSpace(input.License)
	input.Materials = normalizeLines(input.Materials)
	input.Steps = normalizeLines(input.Steps)
	if input.Difficulty == "" {
		input.Difficulty = DifficultyBeginner
	}
	return input
}

// ValidateInput valida os dados de uma prática.
//
// Práticas publicadas precisam de pelo menos uma etapa e de uma licença
// definida; rascunhos podem estar incompletos.
func ValidateInput(input Input, publish bool) error {
	if length := utf8.RuneCountInString(input.Title); length == 0 || length > maxTitleLength {
		return invalid("title", "Informe um título com até 160 caracteres.")
	}
	if utf8.RuneCountInString(input.Summary) > maxSummaryLength {
		return invalid("summary", "O resumo deve ter no máximo 500 caracteres.")
	}
	if utf8.RuneCountInString(input.Category) > maxCategoryLength {
		return invalid("category", "A categoria deve ter no máximo 80 caracteres.")
	}
	if !input.Difficulty.Valid() {
		return invalid("difficulty", "Dificuldade inválida.")
	}
	if utf8.RuneCountInString(input.TimeEstimate) > maxEstimateLength {
		return invalid("time_estimate", "O tempo estimado deve ter no máximo 80 caracteres.")
	}
	if utf8.RuneCountInString(input.CostEstimate) > maxEstimateLength {
		return invalid("cost_estimate", "O custo estimado deve ter no máximo 80 caracteres.")
	}
	if len(input.Materials) > maxMaterials {
		return invalid("materials", "Informe no máximo 50 materiais.")
	}
	for _, material := range input.Materials {
		if length := utf8.RuneCountInString(material); length == 0 || length > maxMaterialLength {
			return invalid("materials", "Cada material deve ter até 200 caracteres.")
		}
	}
	if len(input.Steps) > maxSteps {
		return invalid("steps", "Informe no máximo 50 etapas.")
	}
	for _, step := range input.Steps {
		if length := utf8.RuneCountInString(step); length == 0 || length > maxStepLength {
			return invalid("steps", "Cada etapa deve ter até 1000 caracteres.")
		}
	}
	if utf8.RuneCountInString(input.Safety) > maxSafetyLength {
		return invalid("safety", "Os avisos de segurança devem ter no máximo 2000 caracteres.")
	}
	if utf8.RuneCountInString(input.Adaptations) > maxAdaptationsLength {
		return invalid("adaptations", "As adaptações devem ter no máximo 2000 caracteres.")
	}
	if utf8.RuneCountInString(input.References) > maxReferencesLength {
		return invalid("references", "As referências devem ter no máximo 2000 caracteres.")
	}
	if utf8.RuneCountInString(input.License) > maxLicenseLength {
		return invalid("license", "A licença deve ter no máximo 120 caracteres.")
	}

	if publish {
		if len(input.Steps) == 0 {
			return invalid("steps", "Uma prática publicada precisa ter pelo menos uma etapa.")
		}
		if input.License == "" {
			return invalid("license", "Defina uma licença antes de publicar a prática.")
		}
	}

	return nil
}

func normalizeLines(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}
