package xlspipe

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Sheet names (Russian tabs) for cutover Portugal workbook.
const (
	SheetInputs  = "Ввод"
	SheetDom6    = "Вс 6.09"
	SheetJue3    = "Чт 3.09"
	SheetWedHyp  = "Ср гип"
	SheetSummary = "Сводка"
)

// CutoverPortugalDefaults holds live FO values (2026-08-28).
type CutoverPortugalDefaults struct {
	FerryDom6          float64
	FerryJue3          float64
	FerrySuperiorDelta float64
	HousingLow         float64
	HousingMid         float64
	HousingHigh        float64
	DriveLow           float64
	DriveMid           float64
	DriveHigh          float64
	BoardLow           float64
	BoardMid           float64
	BoardHigh          float64
	FoodLow            float64
	FoodMid            float64
	FoodHigh           float64
	SimLow             float64
	SimMid             float64
	SimHigh            float64
	MonthCap           float64
	ExtraNightsDom6    float64
	ExtraNightsJue3    float64
	ExtraNightsWed     float64
}

// DefaultCutoverPortugal returns FO live snapshot from portugal track (28.08.2026).
func DefaultCutoverPortugal() CutoverPortugalDefaults {
	return CutoverPortugalDefaults{
		FerryDom6:          457.89,
		FerryJue3:          484.09,
		FerrySuperiorDelta: 19.64,
		HousingLow:         509,
		HousingMid:         600,
		HousingHigh:        650,
		DriveLow:           70,
		DriveMid:           85,
		DriveHigh:          100,
		BoardLow:           40,
		BoardMid:           60,
		BoardHigh:          80,
		FoodLow:            150,
		FoodMid:            220,
		FoodHigh:           300,
		SimLow:             50,
		SimMid:             100,
		SimHigh:            150,
		MonthCap:           2500,
		ExtraNightsDom6:    0,
		ExtraNightsJue3:    0,
		ExtraNightsWed:     4,
	}
}

type inputField struct {
	name    string // defined name (ASCII, for formulas)
	label   string
	value   float64
	note    string
	comment string
}

// BuildCutoverPortugalWorkbook creates a multi-sheet cutover budget with Russian labels,
// cell comments on non-obvious inputs, named ranges, and cross-sheet formulas.
func BuildCutoverPortugalWorkbook(d CutoverPortugalDefaults) (*excelize.File, error) {
	f := excelize.NewFile()
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, SheetInputs); err != nil {
		f.Close()
		return nil, err
	}
	for _, name := range []string{SheetDom6, SheetJue3, SheetWedHyp, SheetSummary} {
		if _, err := f.NewSheet(name); err != nil {
			f.Close()
			return nil, err
		}
	}

	if err := writeInputsSheet(f, d); err != nil {
		f.Close()
		return nil, err
	}
	scenarios := []struct {
		sheet, ferry, extra, note string
	}{
		{SheetDom6, "ferry_dom6", "extra_nights_dom6", "Живой слот вс 6.09 20:30; заезд в квартиру вт 8.09"},
		{SheetJue3, "ferry_jue3", "extra_nights_jue3", "Живой чт 3.09; T1a 05–12 если TF-крыша кончается раньше вс"},
		{SheetWedHyp, "ferry_jue3", "extra_nights_wed", "Гипотеза: ср 2.09 20:00 по тарифу Jue3; заезд пт 4.09"},
	}
	for _, sc := range scenarios {
		if err := writeScenarioSheet(f, sc.sheet, sc.ferry, sc.extra, sc.note); err != nil {
			f.Close()
			return nil, err
		}
	}
	if err := writeSummarySheet(f); err != nil {
		f.Close()
		return nil, err
	}
	f.SetActiveSheet(0)
	if err := finalizeWorkbook(f); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func writeInputsSheet(f *excelize.File, d CutoverPortugalDefaults) error {
	if err := setHeaders(f, SheetInputs, "Параметр", "Значение €", "Кратко"); err != nil {
		return err
	}
	fields := []inputField{
		{
			name: "ferry_dom6", label: "Паром вс 6.09 (Básica, без residencia)",
			value: d.FerryDom6, note: "FO live: 2взр+младенец+авто",
			comment: "Fred Olsen Dom 6.09 20:30 SC→Huelva, прибытие Mar 8 09:00. Butaca Normal/Básica без субсидии канарского residencia (−183€). Меняйте после нового live FO.",
		},
		{
			name: "ferry_jue3", label: "Паром чт 3.09 (Básica, без residencia)",
			value: d.FerryJue3, note: "FO live Jue 3 20:00",
			comment: "Прямой рейс 35 ч. Дороже вс на ~26€. Используется также для листа «Ср гип», если своего рейса ср 2.09 нет в продаже.",
		},
		{
			name: "ferry_superior_uplift", label: "Доплата VIP / Butaca Superior",
			value: d.FerrySuperiorDelta, note: "Superior − Normal (Jue3 live)",
			comment: "Разница между Superior и Normal на live Jue3 ≈19,64€. На Dom6 Superior в сессии не перевыбирали — оценка по этой дельте. VIP = salón, не каюта.",
		},
		{
			name: "housing_7n_low", label: "Крыша 7 ночей Setúbal — минимум",
			value: d.HousingLow, note: "Airbnb low band",
			comment: "Короткая аренда T1a (#11): 08–15.09, 1–2BR с парковкой. Низкая граница live Airbnb Setúbal.",
		},
		{
			name: "housing_7n_mid", label: "Крыша 7 ночей Setúbal — целевой mid",
			value: d.HousingMid, note: "Цель #11: 500–650€/нед",
			comment: "Рабочая оценка для брони. Основной столбец mid на листах сценариев.",
		},
		{
			name: "housing_7n_high", label: "Крыша 7 ночей Setúbal — максимум",
			value: d.HousingHigh, note: "Airbnb high band",
		},
		{
			name: "drive_low", label: "Проезд Huelva → Setúbal — мин",
			value: d.DriveLow, note: "OSRM ~3,8 ч",
			comment: "≈342 км: топливо + платные дороги. Низкая/средняя/высокая оценка.",
		},
		{name: "drive_mid", label: "Проезд Huelva → Setúbal — mid", value: d.DriveMid},
		{name: "drive_high", label: "Проезд Huelva → Setúbal — макс", value: d.DriveHigh},
		{
			name: "board_low", label: "Еда на пароме — мин",
			value: d.BoardLow, note: "Меню FO",
			comment: "Питание на борту (Fred Olsen). George 0–3 обычно бесплатно как пассажир — еда отдельно.",
		},
		{name: "board_mid", label: "Еда на пароме — mid", value: d.BoardMid},
		{name: "board_high", label: "Еда на пароме — макс", value: d.BoardHigh},
		{
			name: "food_7d_low", label: "Еда 7 дней в PT — мин",
			value: d.FoodLow, note: "Готовим в apt",
			comment: "Первая неделя в Setúbal после парома — продукты, не рестораны.",
		},
		{name: "food_7d_mid", label: "Еда 7 дней в PT — mid", value: d.FoodMid},
		{name: "food_7d_high", label: "Еда 7 дней в PT — макс", value: d.FoodHigh},
		{
			name: "sim_low", label: "SIM + документы — мин",
			value: d.SimLow, note: "Разовые cutover",
			comment: "eSIM, копии, мелкие госпошлины при cutover. Не включает депозит аренды.",
		},
		{name: "sim_mid", label: "SIM + документы — mid", value: d.SimMid},
		{name: "sim_high", label: "SIM + документы — макс", value: d.SimHigh},
		{
			name: "month_cap", label: "Потолок бюджета на месяц (€)",
			value: d.MonthCap, note: "SoT: 2500€",
			comment: "Жёсткий потолок Sep из source-of-truth. «Остаток» = потолок − итого mid сценария.",
		},
		{
			name: "extra_nights_dom6", label: "Лишние ночи в PT (вс 6.09)",
			value: d.ExtraNightsDom6, note: "0 = TF до вс",
			comment: "Платные ночи в PT до начала 7-дневной крыши. Для Dom6 обычно 0: остаёмся на Тенерифе до вс, заезд вт 8.09.",
		},
		{
			name: "extra_nights_jue3", label: "Лишние ночи в PT (чт 3.09)",
			value: d.ExtraNightsJue3, note: "0 если TF до вс",
			comment: "Если крыша TF кончается раньше вс — нужны ночи 05–07.09 до Airbnb. По умолчанию 0.",
		},
		{
			name: "extra_nights_wed", label: "Лишние ночи в PT (ср гип)",
			value: d.ExtraNightsWed, note: "Гип: заезд пт 4.09",
			comment: "Гипотетический слот ср 2.09 → заезд пт 4.09 = 4 лишних ночи до типичного 7н блока. Рейса ср в FO нет — тариф как Jue3.",
		},
	}
	for i, fld := range fields {
		row := i + 2
		if err := f.SetCellStr(SheetInputs, fmt.Sprintf("A%d", row), fld.label); err != nil {
			return err
		}
		if err := defineInput(f, SheetInputs, fld.name, row, fld.value, fld.note, fld.comment); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(SheetInputs, "A25", "—"); err != nil {
		return err
	}
	if err := f.SetCellStr(SheetInputs, "B25", "Редактируйте жёлтые ячейки"); err != nil {
		return err
	}
	if err := f.SetCellStr(SheetInputs, "C25", "Формулы на листах сценариев и «Сводка» пересчитаются в OnlyOffice"); err != nil {
		return err
	}
	return styleSheet(f, SheetInputs, SheetInputs)
}

func writeScenarioSheet(f *excelize.File, sheet, ferryName, extraNightsName, scenarioNote string) error {
	if err := setHeaders(f, sheet, "Статья", "Мин €", "Mid €", "Макс €", "Пояснение"); err != nil {
		return err
	}
	if err := f.SetCellStr(sheet, "A2", "Паром Básica"); err != nil {
		return err
	}
	for col, ref := range []string{ferryName, ferryName, ferryName} {
		cell, _ := excelize.CoordinatesToCellName(col+2, 2)
		if err := f.SetCellFormula(sheet, cell, "="+ref); err != nil {
			return err
		}
	}
	if err := addCellComment(f, sheet, "A2", "Тариф парома для этого сценария. Берётся с листа «Ввод»."); err != nil {
		return err
	}
	lines := []struct {
		label                string
		low, mid, high, note string
		comment              string
	}{
		{"Крыша 7 н Setúbal", "housing_7n_low", "housing_7n_mid", "housing_7n_high", "Airbnb T1a #11", ""},
		{"Huelva → Setúbal", "drive_low", "drive_mid", "drive_high", "OSRM ~3,8 ч", ""},
		{"Еда на борту", "board_low", "board_mid", "board_high", "Меню FO", ""},
		{"Еда 7 д в PT", "food_7d_low", "food_7d_mid", "food_7d_high", "Готовим дома", ""},
		{"SIM / документы", "sim_low", "sim_mid", "sim_high", "Cutover", ""},
	}
	for i, ln := range lines {
		row := i + 3
		if err := setFormulaRow(f, sheet, row, ln.label,
			"="+ln.low, "="+ln.mid, "="+ln.high, ln.note); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(sheet, "A8", "Итого Básica"); err != nil {
		return err
	}
	for col := 2; col <= 4; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 8)
		colL, _ := excelize.CoordinatesToCellName(col, 2)
		colH, _ := excelize.CoordinatesToCellName(col, 7)
		if err := f.SetCellFormula(sheet, cell, fmt.Sprintf("=SUM(%s:%s)", colL, colH)); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(sheet, "E8", "SUM строк 2–7"); err != nil {
		return err
	}
	if err := setFormulaRow(f, sheet, 9, "Итого Superior",
		"=B8+ferry_superior_uplift", "=C8+ferry_superior_uplift", "=D8+ferry_superior_uplift",
		"Básica + VIP"); err != nil {
		return err
	}
	if err := addCellComment(f, sheet, "A9", "Butaca Superior / VIP salón. Доплата с листа «Ввод»."); err != nil {
		return err
	}
	if err := f.SetCellStr(sheet, "A10", "Лишние ночи PT"); err != nil {
		return err
	}
	for col, housing := range []string{"housing_7n_low", "housing_7n_mid", "housing_7n_high"} {
		cell, _ := excelize.CoordinatesToCellName(col+2, 10)
		formula := fmt.Sprintf("=%s*%s/7", extraNightsName, housing)
		if err := f.SetCellFormula(sheet, cell, formula); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(sheet, "E10", "ночей × (крыша/7)"); err != nil {
		return err
	}
	if err := addCellComment(f, sheet, "A10", "Платное жильё до начала 7-дневной брони. Число ночей — на листе «Ввод» для этого сценария."); err != nil {
		return err
	}
	if err := setFormulaRow(f, sheet, 11, "Итого с ночами",
		"=B8+B10", "=C8+C10", "=D8+D10", scenarioNote); err != nil {
		return err
	}
	if err := f.SetCellStr(sheet, "A12", "Остаток от потолка"); err != nil {
		return err
	}
	for _, col := range []string{"B", "C", "D"} {
		if err := f.SetCellFormula(sheet, col+"12", "=month_cap-C11"); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(sheet, "E12", "потолок − mid итого"); err != nil {
		return err
	}
	if err := addCellComment(f, sheet, "C12", "Сколько остаётся от месячного потолка 2500€ после cutover (mid)."); err != nil {
		return err
	}
	if err := f.SetCellStr(sheet, "A13", "Среднее по строкам"); err != nil {
		return err
	}
	for col := 2; col <= 4; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 13)
		colL, _ := excelize.CoordinatesToCellName(col, 2)
		colH, _ := excelize.CoordinatesToCellName(col, 7)
		if err := f.SetCellFormula(sheet, cell, fmt.Sprintf("=AVERAGE(%s:%s)", colL, colH)); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(sheet, "E13", "AVG статей 2–7"); err != nil {
		return err
	}
	return styleSheet(f, sheet, SheetInputs)
}

func writeSummarySheet(f *excelize.File) error {
	if err := setHeaders(f, SheetSummary,
		"Сценарий", "Básica mid", "Superior mid", "Итого mid", "Остаток", "Δ vs вс"); err != nil {
		return err
	}
	rows := []struct {
		label, sheet string
	}{
		{"Вс 6.09 (live)", SheetDom6},
		{"Чт 3.09 (live)", SheetJue3},
		{"Ср 2.09 (гипотеза)", SheetWedHyp},
	}
	qs := quoteSheet
	for i, r := range rows {
		row := i + 2
		if err := f.SetCellStr(SheetSummary, fmt.Sprintf("A%d", row), r.label); err != nil {
			return err
		}
		pairs := []struct {
			col int
			ref string
		}{
			{2, fmt.Sprintf("%s!C8", qs(r.sheet))},
			{3, fmt.Sprintf("%s!C9", qs(r.sheet))},
			{4, fmt.Sprintf("%s!C11", qs(r.sheet))},
			{5, fmt.Sprintf("%s!C12", qs(r.sheet))},
		}
		for _, p := range pairs {
			cell, _ := excelize.CoordinatesToCellName(p.col, row)
			if err := f.SetCellFormula(SheetSummary, cell, "="+p.ref); err != nil {
				return err
			}
		}
		cell, _ := excelize.CoordinatesToCellName(6, row)
		if err := f.SetCellFormula(SheetSummary, cell, fmt.Sprintf("=D%d-$D$2", row)); err != nil {
			return err
		}
	}
	if err := f.SetCellStr(SheetSummary, "A6", "Потолок месяца"); err != nil {
		return err
	}
	if err := f.SetCellFormula(SheetSummary, "D6", "=month_cap"); err != nil {
		return err
	}
	if err := f.SetCellStr(SheetSummary, "A7", "Лучший итого mid"); err != nil {
		return err
	}
	if err := f.SetCellFormula(SheetSummary, "D7", "=MIN(D2:D4)"); err != nil {
		return err
	}
	if err := f.SetCellStr(SheetSummary, "E7", "MIN по сценариям"); err != nil {
		return err
	}
	if err := addCellComment(f, SheetSummary, "D7", "Минимальный mid «Итого с ночами» среди трёх сценариев. Сейчас обычно вс 6.09."); err != nil {
		return err
	}
	return styleSheet(f, SheetSummary, SheetInputs)
}
