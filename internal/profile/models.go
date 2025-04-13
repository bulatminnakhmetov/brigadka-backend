package profile

// TranslatedItem представляет элемент справочника с переводами
type TranslatedItem struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Справочники для профилей
type ActivityTypeCatalog []TranslatedItem
type ImprovStyleCatalog []TranslatedItem
type ImprovGoalCatalog []TranslatedItem
type MusicGenreCatalog []TranslatedItem
type MusicInstrumentCatalog []TranslatedItem
