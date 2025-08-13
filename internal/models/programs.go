package models

// ProgramCategory represents a category of driving programs
type ProgramCategory struct {
	Name     string   `json:"name"`
	Programs []string `json:"programs"`
}

// AllPrograms contains all available BMW Driving Center programs
var AllPrograms = []ProgramCategory{
	{
		Name: "Experience Programs",
		Programs: []string{
			"Test Drive",
			"Off-Road",
			"Taxi",
			"i Drive",
			"Night Drive",
			"Scenic Drive",
			"On-Road",
			"X-Bus",
		},
	},
	{
		Name: "Training Programs",
		Programs: []string{
			"Starter Pack",
			"i Starter Pack",
			"MINI Starter Pack",
			"M Core",
			"BEV Core",
			"Intensive",
			"M Intensive",
			"JCW Intensive",
			"M Drift I",
			"M Drift II",
			"M Drift III",
		},
	},
	{
		Name: "Owner Programs",
		Programs: []string{
			"Owners Track Day",
			"Owners Drift Day",
		},
	},
	{
		Name: "Junior Campus Programs",
		Programs: []string{
			"Laboratory",
			"Workshop",
		},
	},
}

// GetAllProgramNames returns a flat list of all program names
func GetAllProgramNames() []string {
	var programs []string
	for _, category := range AllPrograms {
		programs = append(programs, category.Programs...)
	}
	return programs
}

// GetKoreanProgramNames returns Korean names for programs (mapping)
var ProgramNameMap = map[string]string{
	"Test Drive":          "테스트 드라이브",
	"Off-Road":           "오프로드",
	"Taxi":               "택시",
	"i Drive":            "i 드라이브",
	"Night Drive":        "나이트 드라이브",
	"Scenic Drive":       "시닉 드라이브",
	"On-Road":            "온로드",
	"X-Bus":              "X-버스",
	"Starter Pack":       "스타터 팩",
	"i Starter Pack":     "i 스타터 팩",
	"MINI Starter Pack":  "MINI 스타터 팩",
	"M Core":             "M 코어",
	"BEV Core":           "BEV 코어",
	"Intensive":          "인텐시브",
	"M Intensive":        "M 인텐시브",
	"JCW Intensive":      "JCW 인텐시브",
	"M Drift I":          "M 드리프트 I",
	"M Drift II":         "M 드리프트 II",
	"M Drift III":        "M 드리프트 III",
	"Owners Track Day":   "오너스 트랙 데이",
	"Owners Drift Day":   "오너스 드리프트 데이",
	"Laboratory":         "연구실",
	"Workshop":           "워크샵",
}