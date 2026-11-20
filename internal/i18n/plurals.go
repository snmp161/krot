package i18n

type pluralCategory int

const (
	pluralZero pluralCategory = iota
	pluralOne
	pluralTwo
	pluralFew
	pluralMany
	pluralOther
)

type pluralForms struct {
	zero  string
	one   string
	two   string
	few   string
	many  string
	other string
}

var pluralCatalogs = map[Language]map[string]pluralForms{
	English: {
		"client": {one: "client", other: "clients"},
	},
	Russian: {
		"client": {one: "клиент", few: "клиента", many: "клиентов", other: "клиентов"},
	},
	ChineseSimplified: {
		"client": {other: "客户端"},
	},
	ChineseTraditional: {
		"client": {other: "客戶端"},
	},
	Spanish: {
		"client": {one: "cliente", other: "clientes"},
	},
	Portuguese: {
		"client": {one: "cliente", other: "clientes"},
	},
	Korean: {
		"client": {other: "클라이언트"},
	},
	Japanese: {
		"client": {other: "クライアント"},
	},
	Arabic: {
		"client": {zero: "عملاء", one: "عميل", two: "عميلان", few: "عملاء", many: "عميلًا", other: "عميل"},
	},
	French: {
		"client": {one: "client", other: "clients"},
	},
	German: {
		"client": {one: "Client", other: "Clients"},
	},
}

func Plural(n int, key string) string {
	language := Current()
	forms := pluralCatalogs[language][key]
	if forms.other == "" {
		forms = pluralCatalogs[English][key]
	}
	if forms.other == "" {
		return key
	}
	var translated string
	switch pluralCategoryFor(language, n) {
	case pluralZero:
		translated = forms.zero
	case pluralOne:
		translated = forms.one
	case pluralTwo:
		translated = forms.two
	case pluralFew:
		translated = forms.few
	case pluralMany:
		translated = forms.many
	default:
		translated = forms.other
	}
	if translated == "" {
		return forms.other
	}
	return translated
}

func pluralCategoryFor(language Language, n int) pluralCategory {
	if n < 0 {
		n = -n
	}
	switch language {
	case Arabic:
		switch {
		case n == 0:
			return pluralZero
		case n == 1:
			return pluralOne
		case n == 2:
			return pluralTwo
		case n%100 >= 3 && n%100 <= 10:
			return pluralFew
		case n%100 >= 11 && n%100 <= 99:
			return pluralMany
		default:
			return pluralOther
		}
	case Russian:
		switch {
		case n%10 == 1 && n%100 != 11:
			return pluralOne
		case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
			return pluralFew
		default:
			return pluralMany
		}
	case French, Portuguese:
		if n == 0 || n == 1 {
			return pluralOne
		}
	case ChineseSimplified, ChineseTraditional, Korean, Japanese:
		return pluralOther
	default:
		if n == 1 {
			return pluralOne
		}
	}
	return pluralOther
}
