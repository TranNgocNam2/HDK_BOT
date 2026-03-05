package service

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"hkd.nam2507/model"
	"strconv"
	"strings"
	"time"
)

var cityMap = map[string]string{
	"An Giang":    "133",
	"Bắc Ninh":    "100",
	"Cà Mau":      "143",
	"Cần Thơ":     "138",
	"Cao Bằng":    "90",
	"Đà Nẵng":     "112",
	"Đắk Lắk":     "120",
	"Điện Biên":   "102",
	"Đồng Nai":    "128",
	"Đồng Tháp":   "132",
	"Gia Lai":     "119",
	"Hà Nội":      "81",
	"Hà Tĩnh":     "108",
	"Hải Phòng":   "82",
	"Hồ Chí Minh": "122",
	"Huế":         "144",
	"Hưng Yên":    "84",
	"Khánh Hòa":   "117",
	"Lai Châu":    "103",
	"Lâm Đồng":    "123",
	"Lạng Sơn":    "93",
	"Lào Cai":     "91",
	"Nghệ An":     "107",
	"Ninh Bình":   "88",
	"Phú Thọ":     "97",
	"Quảng Ngãi":  "114",
	"Quảng Ninh":  "101",
	"Quảng Trị":   "110",
	"Sơn La":      "104",
	"Tây Ninh":    "126",
	"Thái Nguyên": "96",
	"Thanh Hoá":   "106",
	"Tuyên Quang": "94",
	"Vĩnh Long":   "135",
}

func connectBrowser() (context.Context, context.CancelFunc, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(
		context.Background(),
		"ws://localhost:9222",
	)

	ctx, ctxCancel := chromedp.NewContext(allocCtx)

	cancel := func() {
		ctxCancel()
		allocCancel()
	}

	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("không thể kết nối Chrome: %w", err)
	}

	return ctx, cancel, nil
}

func runSteps(ctx context.Context, steps []struct {
	name   string
	action chromedp.Action
}) error {
	for _, step := range steps {
		fmt.Println("▶️ Đang chạy bước:", step.name)
		stepCtx, stepCancel := context.WithTimeout(ctx, 15*time.Second)
		err := chromedp.Run(stepCtx, step.action)
		stepCancel()
		if err != nil {
			return fmt.Errorf("kẹt tại [%s]: %w", step.name, err)
		}
		fmt.Println("✅ Xong:", step.name)
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func SubmitHoSo(data model.Hokinhdoanh) error {
	ctx, cancel, err := connectBrowser()
	if err != nil {
		return err
	}
	defer cancel()

	const baseURL = "https://hokinhdoanh.dkkd.gov.vn/HkdOnline"

	steps := []struct {
		name   string
		action chromedp.Action
	}{
		{"Navigate", chromedp.Navigate(baseURL + "/Default.aspx")},
		{"WaitMenu", chromedp.WaitVisible(`a.AspNet-Menu-Link`, chromedp.ByQuery)},
		{"ClickMenu", chromedp.Evaluate(`__doPostBack('ctl00$LV3$mCon','bĐăng ký Hộ kinh doanh')`, nil)},
		{"WaitNextBtn", chromedp.WaitVisible(`#ctl00_C_myWizard_StepNavigationTemplateContainerID_StepNextButton`)},
		{"ClickNext", chromedp.Click(`#ctl00_C_myWizard_StepNavigationTemplateContainerID_StepNextButton`)},
		{"WaitStart", chromedp.WaitVisible(`#ctl00_C_myWizard_FinishNavigationTemplateContainerID_FinishButton`)},
		{"ClickStart", chromedp.Click(`#ctl00_C_myWizard_FinishNavigationTemplateContainerID_FinishButton`)},
		{"WaitHinhThuc", chromedp.WaitVisible(`#ctl00_C_BLCtl_CtlList_ctl02_LnkEdit`)},
		{"ClickHinhThuc", chromedp.Click(`#ctl00_C_BLCtl_CtlList_ctl02_LnkEdit`)},
		{"WaitSave", chromedp.WaitVisible(`#ctl00_C_btnSave`)},
		{"ClickSave", chromedp.Click(`#ctl00_C_btnSave`)},
	}

	if err := runSteps(ctx, steps); err != nil {
		return err
	}

	if err := submitDiaChi(ctx, data.DiaChiKinhDoanh, data.SDT); err != nil {
		return err
	}

	if err := submitNganhNghe(ctx, data.NganhNgheKinhDoanh); err != nil {
		return err
	}

	if err := submitTenHKD(ctx, data.TenHoKinhDoanh); err != nil {
		return err
	}
	return nil
}

func submitDiaChi(ctx context.Context, diaChi model.DiaChi, sdt string) error {
	cityID, ok := cityMap[normalizeCityName(diaChi.TinhTP)]
	if !ok {
		return fmt.Errorf("không tìm thấy tỉnh/TP: %s", diaChi.TinhTP)
	}

	steps := []struct {
		name   string
		action chromedp.Action
	}{
		{"ClickDiaChi", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.WaitVisible(`#ctl00_C_BLCtl_CtlList_ctl03_LnkEdit`),
				chromedp.Click(`#ctl00_C_BLCtl_CtlList_ctl03_LnkEdit`),
				chromedp.Sleep(2*time.Second),
			)
		})},
		{"WaitCity", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.WaitReady(`body`),
				chromedp.Sleep(2*time.Second),
				chromedp.WaitVisible(`#ctl00_C_ADDRCtl_CITY_IDFld`, chromedp.ByID),
			)
		})},
		{"SelectCity", chromedp.SetValue(`#ctl00_C_ADDRCtl_CITY_IDFld`, cityID, chromedp.ByID)},
		{"TriggerCity", chromedp.Evaluate(`
			var el = document.getElementById('ctl00_C_ADDRCtl_CITY_IDFld');
			el.dispatchEvent(new Event('change'));
		`, nil)},

		{"SelectWard", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				// Nhập vào input autocomplete (không phải select)
				chromedp.WaitVisible(`input.ui_combobox_input`),
				chromedp.Clear(`input.ui_combobox_input`),
				chromedp.SendKeys(`input.ui_combobox_input`, diaChi.XaPhuong),
				chromedp.Sleep(1*time.Second),
				// Click vào option xuất hiện trong dropdown
				chromedp.WaitVisible(`ul.ui-autocomplete li`),
				chromedp.Click(`ul.ui-autocomplete li:first-child`),
			)
		})},
		{"TriggerWard", chromedp.Evaluate(`
			var el = document.getElementById('ctl00_C_ADDRCtl_WARD_IDFld');
			el.dispatchEvent(new Event('change'));
		`, nil)},
		{"WaitStreet", chromedp.WaitVisible(`#ctl00_C_ADDRCtl_STREET_NUMBERFld`)},
		{"InputStreet", chromedp.SetValue(`#ctl00_C_ADDRCtl_STREET_NUMBERFld`, diaChi.SoNha, chromedp.ByID)},
		{"InputPhone", chromedp.SetValue(`#ctl00_C_DCONTClt_HO_PHONEFld`, sdt, chromedp.ByID)},
		// Thêm bước TroVe vào cuối submitDiaChi, trước ClickSave
		{"TroVe", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.Click(`#ctl00_C_btnSave`),
				chromedp.Sleep(2*time.Second),
				// Sau khi lưu, bấm Trở về để về trang danh sách
				chromedp.WaitVisible(`#ctl00_C_btnBack`),
				chromedp.Click(`#ctl00_C_btnBack`),
				chromedp.Sleep(2*time.Second),
			)
		})},
	}

	return runSteps(ctx, steps)
}

func submitNganhNghe(ctx context.Context, nganhNgheList []model.NganhNgheKinhDoanh) error {
	steps := []struct {
		name   string
		action chromedp.Action
	}{
		{"ClickNganhNghe", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.Navigate(`https://hokinhdoanh.dkkd.gov.vn/HkdOnline/Forms/APP/DW_APPLICATIONEdit.aspx`),
				chromedp.WaitReady(`body`),
				chromedp.Sleep(2*time.Second),
				chromedp.WaitVisible(`#ctl00_C_BLCtl_CtlList_ctl04_LnkEdit`),
				chromedp.Click(`#ctl00_C_BLCtl_CtlList_ctl04_LnkEdit`),
				chromedp.WaitReady(`body`),
				chromedp.Sleep(3*time.Second),
			)
		})},
	}

	for i, nn := range nganhNgheList {
		maNganh := strconv.Itoa(nn.MaNganh)
		idx := i

		steps = append(steps,
			struct {
				name   string
				action chromedp.Action
			}{
				fmt.Sprintf("InputMaNganh_%d", idx+1),
				chromedp.ActionFunc(func(ctx context.Context) error {
					return chromedp.Run(ctx,
						chromedp.Evaluate(fmt.Sprintf(`
							var el = document.getElementById('ctl00_C_newBusinessLineCode');
							el.value = '%s';
							el.dispatchEvent(new Event('input'));
							el.dispatchEvent(new Event('change'));
						`, maNganh), nil),
						chromedp.Sleep(1*time.Second),
					)
				}),
			},
			struct {
				name   string
				action chromedp.Action
			}{
				fmt.Sprintf("ClickThem_%d", idx+1),
				chromedp.ActionFunc(func(ctx context.Context) error {
					return chromedp.Run(ctx,
						chromedp.Click(`#ctl00_C_BtnAddBl`),
						chromedp.Sleep(2*time.Second),
					)
				}),
			},
		)
	}

	steps = append(steps,
		struct {
			name   string
			action chromedp.Action
		}{
			"TickNganhChinh",
			chromedp.ActionFunc(func(ctx context.Context) error {
				return chromedp.Run(ctx,
					chromedp.WaitReady(`body`),
					chromedp.Sleep(2*time.Second),
					chromedp.WaitVisible(`input[name="ismain"]`),
					chromedp.Click(`input[name="ismain"]`),
				)
			}),
		},
		struct {
			name   string
			action chromedp.Action
		}{
			"ClickCapNhatChinh",
			chromedp.ActionFunc(func(ctx context.Context) error {
				return chromedp.Run(ctx,
					chromedp.Click(`#ctl00_C_btnUpdateMain`),
					chromedp.Sleep(1*time.Second),
				)
			}),
		},
		struct {
			name   string
			action chromedp.Action
		}{
			"ClickLuu",
			chromedp.ActionFunc(func(ctx context.Context) error {
				return chromedp.Run(ctx,
					chromedp.Click(`#ctl00_C_BtnSaveNotVSIC`),
					chromedp.Sleep(2*time.Second),
				)
			}),
		},
		struct {
			name   string
			action chromedp.Action
		}{
			"TroVe",
			chromedp.ActionFunc(func(ctx context.Context) error {
				return chromedp.Run(ctx,
					chromedp.WaitVisible(`#ctl00_C_btnBack`),
					chromedp.Click(`#ctl00_C_btnBack`),
					chromedp.Sleep(2*time.Second),
				)
			}),
		},
	)

	return runSteps(ctx, steps)
}

func submitTenHKD(ctx context.Context, tenHKD string) error {
	steps := []struct {
		name   string
		action chromedp.Action
	}{
		{"ClickTenHKD", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.WaitVisible(`#ctl00_C_BLCtl_CtlList_ctl05_LnkEdit`),
				chromedp.Click(`#ctl00_C_BLCtl_CtlList_ctl05_LnkEdit`),
				chromedp.WaitReady(`body`),
				chromedp.Sleep(2*time.Second),
			)
		})},
		{"InputTenHKD", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.Sleep(2*time.Second),
				chromedp.Evaluate(fmt.Sprintf(`
            var el = document.getElementById('ctl00_C_NAMEFld');
            el.value = '%s';
            el.dispatchEvent(new Event('input'));
            el.dispatchEvent(new Event('change'));
        `, strings.ToUpper(tenHKD)), nil),
			)
		})},
		{"ClickLuu", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.Click(`#ctl00_C_BtnSave`),
				chromedp.Sleep(2*time.Second),
			)
		})},
		{"TroVe", chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Run(ctx,
				chromedp.WaitVisible(`#ctl00_C_btnBack`),
				chromedp.Click(`#ctl00_C_btnBack`),
				chromedp.Sleep(2*time.Second),
			)
		})},
	}

	return runSteps(ctx, steps)
}

func normalizeCityName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer(
		"Thành phố ", "",
		"Tỉnh ", "",
		"TP. ", "",
		"TP ", "",
		"tp. ", "",
		"tp ", "",
	)
	return replacer.Replace(name)
}
