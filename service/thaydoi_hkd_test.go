package service

import (
	"archive/zip"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"hkd.nam2507/model"
)

func TestCheckVonPlaceholders(t *testing.T) {
	p := "../templates/1.docx"
	r, err := zip.OpenReader(p)
	if err != nil {
		t.Fatalf("%s: error %v", p, err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			s := string(b)

			idx := strings.Index(s, "ten_nganh")
			if idx != -1 {
				trStart1 := strings.LastIndex(s[:idx], "<w:tr")
				idx1 := strings.LastIndex(s[:idx], "<w:tr ")
				idx2 := strings.LastIndex(s[:idx], "<w:tr>")
				trStart2 := idx1
				if idx2 > trStart2 {
					trStart2 = idx2
				}
				t.Logf("ten_nganh: old trStart=%d, new trStart=%d", trStart1, trStart2)
				t.Logf("Tag at old trStart: %s", s[trStart1:trStart1+20])
				t.Logf("Tag at new trStart: %s", s[trStart2:trStart2+20])
			}
		}
	}
}

func TestFillUyQuyenTemplate(t *testing.T) {
	data := model.UyQuyenData{
		HoVaTen:         "Nguyễn Thị Huỳnh Như",
		NgaySinh:        "15/08/1995",
		CCCD:            "079195001234",
		NgayCap:         "10/10/2021",
		CoQuanCap:       "Cục Cảnh sát QLHC về TTXH",
		DiaChiThuongTru: "123 Đường Số 7, Phường Bình Trị Đông B, Quận Bình Tân, TP.HCM",
		CoQuan:          "Phường Bình Tân",
	}

	outPath, err := FillUyQuyenTemplate(data, "../templates/uyquyen_thaydoi.docx")
	if err != nil {
		t.Fatalf("FillUyQuyenTemplate failed: %v", err)
	}
	defer os.Remove(outPath)

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("zip.OpenReader failed: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)
			t.Logf("Checking NgaySinh in docx: contains ngaysinh=%v, contains 15/08/1995=%v", strings.Contains(xmlStr, "ngaysinh"), strings.Contains(xmlStr, "15/08/1995"))
			idxNs := strings.Index(xmlStr, "Ngày sinh:")
			if idxNs != -1 {
				t.Logf("After Ngày sinh: in docx: %s", xmlStr[idxNs:idxNs+200])
			}
			if !strings.Contains(xmlStr, "NGUYỄN THỊ HUỲNH NHƯ") {
				t.Errorf("Document missing uppercase name")
			}
			if !strings.Contains(xmlStr, "079195001234") {
				t.Errorf("Document missing CCCD")
			}
			if !strings.Contains(xmlStr, "15/08/1995") {
				t.Errorf("Document missing NgaySinh")
			}
			if strings.Contains(xmlStr, "{hovaten}") || strings.Contains(xmlStr, "{cccd}") {
				t.Errorf("Document still contains placeholders!")
			}
		}
	}
}

func TestFillThayDoiTemplateTopHeader(t *testing.T) {
	data := model.ThayDoiHKD{
		HoVaTen:   "Nguyễn Văn A",
		TenHKD:    "HỘ KINH DOANH BÁNH ƯỚT KHƯƠNG VĂN TÙNG",
		MaHKD:     "033083002399",
		CoQuan:    "Phường Bình Tân",
		CapDoiGCN: false,
	}

	outPath, err := FillThayDoiTemplate(data, "../templates/1.docx")
	if err != nil {
		t.Fatalf("FillThayDoiTemplate error: %v", err)
	}
	defer os.Remove(outPath)

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("zip.OpenReader error: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)
			t.Logf("Checking NgaySinh in docx: contains ngaysinh=%v, contains 15/08/1995=%v", strings.Contains(xmlStr, "ngaysinh"), strings.Contains(xmlStr, "15/08/1995"))
			idxNs := strings.Index(xmlStr, "Ngày sinh:")
			if idxNs != -1 {
				t.Logf("After Ngày sinh: in docx: %s", xmlStr[idxNs:idxNs+200])
			}
			if strings.Contains(xmlStr, "ten_hkd") {
				t.Errorf("Document still contains 'ten_hkd' placeholder!")
			}
			if strings.Contains(xmlStr, "{sonha}") {
				t.Errorf("Document still contains '{sonha}' placeholder!")
			}
			if strings.Contains(xmlStr, "{solaodong}") {
				t.Errorf("Document still contains '{solaodong}' placeholder!")
			}
			if strings.Contains(xmlStr, "{stt_sua}") {
				t.Errorf("Document still contains '{stt_sua}' placeholder!")
			}
			if strings.Contains(xmlStr, "{ma_sua}") {
				t.Errorf("Document still contains '{ma_sua}' placeholder!")
			}
			if !strings.Contains(xmlStr, "Thành phố Hồ Chí Minh") {
				t.Errorf("Document does not contain default 'Thành phố Hồ Chí Minh'!")
			}
			if !strings.Contains(xmlStr, "BÁNH ƯỚT KHƯƠNG VĂN TÙNG") {
				t.Errorf("Document does not contain 'BÁNH ƯỚT KHƯƠNG VĂN TÙNG'!")
			}
		}
	}
}

func TestFillThayDoiTemplateFull(t *testing.T) {
	data := model.ThayDoiHKD{
		HoVaTen: "Nguyễn Văn A",
		TenHKD:  "HỘ KINH DOANH HOA MAI",
		MaHKD:   "0319988776",
		CoQuan:  "Phường Bình Tân",
		ThayDoiTruSo: &model.ThayDoiTruSo{
			SoNha:    "123 Đường Số 1, KP2",
			CoQuan:   "Phường Bình Tân",
			ThanhPho: "Thành phố Hồ Chí Minh",
			SDT:      "0901234567",
		},
		ThayDoiNganhNghe: &model.ThayDoiNganhNghe{
			BoSung: []model.NganhNgheKinhDoanh{
				{
					TenNganh:     "Sản xuất các loại bánh từ bột",
					MaNganh:      1071,
					ChiTiet:      "Sản xuất bánh ướt",
					LaNganhChinh: true,
				},
				{
					TenNganh:     "Bán lẻ thực phẩm",
					MaNganh:      4711,
					LaNganhChinh: false,
				},
			},
			Bo: []model.NganhNgheKinhDoanh{
				{
					TenNganh: "Nhà hàng và các dịch vụ ăn uống phục vụ lưu động",
					MaNganh:  5610,
				},
			},
		},
		CapDoiGCN: true,
	}

	outPath, err := FillThayDoiTemplate(data, "../templates/1.docx")
	if err != nil {
		t.Fatalf("FillThayDoiTemplate error: %v", err)
	}
	defer os.Remove(outPath)

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("zip.OpenReader error: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)
			t.Logf("Checking NgaySinh in docx: contains ngaysinh=%v, contains 15/08/1995=%v", strings.Contains(xmlStr, "ngaysinh"), strings.Contains(xmlStr, "15/08/1995"))
			idxNs := strings.Index(xmlStr, "Ngày sinh:")
			if idxNs != -1 {
				t.Logf("After Ngày sinh: in docx: %s", xmlStr[idxNs:idxNs+200])
			}
			t.Logf("Checking final: len=%d, 5610=%v, 1071=%v, sonha=%v", len(xmlStr), strings.Contains(xmlStr, "5610"), strings.Contains(xmlStr, "1071"), strings.Contains(xmlStr, "123 Đường Số 1"))
			if !strings.Contains(xmlStr, "5610") {
				// Search where bo is
				t.Errorf("5610 NOT in output docx! Index of Bo: %d", strings.Index(xmlStr, "nganhnghe_bo"))
			}
		}
	}
}

func TestThayDoiNewFeatures(t *testing.T) {
	// 1. Kiểm tra parse ngành nghề với format STT | Mã | Chi tiết | (chính)
	inputBs := "1 | 1071 | Sản xuất các loại bánh ngọt | (chính)\n2 | 4711 | Bán lẻ lương thực"
	parsedBs := parseNganhNgheWithChiTiet(inputBs)
	if len(parsedBs) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(parsedBs))
	}
	if parsedBs[0].MaNganh != 1071 || !parsedBs[0].LaNganhChinh || parsedBs[0].ChiTiet != "Sản xuất các loại bánh ngọt" {
		t.Errorf("Item 0 mismatch: %+v", parsedBs[0])
	}
	if parsedBs[1].MaNganh != 4711 || parsedBs[1].LaNganhChinh || parsedBs[1].ChiTiet != "Bán lẻ lương thực" {
		t.Errorf("Item 1 mismatch: %+v", parsedBs[1])
	}

	// 2. Kiểm tra parse thay đổi vốn
	inputVon := "150.000.000 VNĐ - Tăng vốn (từ ngày 15/05/2026)"
	parsedVon := parseThayDoiVon(inputVon)
	if parsedVon == nil {
		t.Fatalf("parseThayDoiVon returned nil")
	}
	if parsedVon.VonKinhDoanhMoi != "150.000.000 VNĐ" {
		t.Errorf("VonKinhDoanhMoi mismatch: %s", parsedVon.VonKinhDoanhMoi)
	}
	if parsedVon.HinhThucThayDoi != "Tăng vốn" {
		t.Errorf("HinhThucThayDoi mismatch: %s", parsedVon.HinhThucThayDoi)
	}
	if parsedVon.ThoiGianThayDoi != "15/05/2026" {
		t.Errorf("ThoiGianThayDoi mismatch: %s", parsedVon.ThoiGianThayDoi)
	}

	// 3. Kiểm tra thay thế template có ThongTinThem và không Cấp đổi
	data := model.ThayDoiHKD{
		HoVaTen: "Lê Văn Tám",
		TenHKD:  "HỘ KINH DOANH HOA TƯƠI ĐÀ LẠT",
		MaHKD:   "0319888999",
		CoQuan:  "Phường 5",
		ThayDoiTruSo: &model.ThayDoiTruSo{
			SoNha: "456 Đường Nguyễn Huệ",
			SDT:   "0988776655",
		},
		ThongTinThem: "Tầng 5, Tòa nhà Bitexco",
		ThayDoiVon:   parsedVon,
		CapDoiGCN:    false,
	}

	outPath, err := FillThayDoiTemplate(data, "../templates/1.docx")
	if err != nil {
		t.Fatalf("FillThayDoiTemplate error: %v", err)
	}
	defer os.Remove(outPath)

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("zip.OpenReader error: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)
			t.Logf("Checking NgaySinh in docx: contains ngaysinh=%v, contains 15/08/1995=%v", strings.Contains(xmlStr, "ngaysinh"), strings.Contains(xmlStr, "15/08/1995"))
			idxNs := strings.Index(xmlStr, "Ngày sinh:")
			if idxNs != -1 {
				t.Logf("After Ngày sinh: in docx: %s", xmlStr[idxNs:idxNs+200])
			}
			posTTT := strings.Index(xmlStr, "Bitexco")
			t.Logf("Bitexco index in output docx: %d", posTTT)
			if posTTT == -1 {
				posOld := strings.Index(xmlStr, "thongtinthem")
				t.Logf("thongtinthem index in output docx: %d", posOld)
				if posOld != -1 {
					t.Logf("Around thongtinthem in output: %s", xmlStr[posOld-100:posOld+100])
				}
			}
			if strings.Contains(xmlStr, "{thongtinthem}") {
				t.Errorf("Document still contains '{thongtinthem}' placeholder!")
			}
			if strings.Contains(xmlStr, "{solaodong}") {
				t.Errorf("Document still contains '{solaodong}'!")
			}
			if strings.Contains(xmlStr, "{stt_sua}") || strings.Contains(xmlStr, "{ma_sua}") {
				t.Errorf("Document still contains '{stt_sua}' or '{ma_sua}'!")
			}
			if !strings.Contains(xmlStr, "150.000.000 VNĐ") {
				t.Errorf("Document does not contain new capital!")
			}

			// Kiểm tra phần Cấp đổi: vì CapDoiGCN=false, các trường ở dưới phải là format ....
			posCapDoi := strings.Index(xmlStr, "ĐĂNG KÝ CẤP ĐỔI SANG GIẤY CHỨNG NHẬN")
			if posCapDoi != -1 {
				partLower := xmlStr[posCapDoi:]
				if strings.Contains(partLower, "HOA TƯƠI ĐÀ LẠT") {
					t.Errorf("Lower part should NOT contain HKD name when CapDoiGCN=false!")
				}
				if strings.Contains(partLower, "456 Đường Nguyễn Huệ") {
					t.Errorf("Lower part should NOT contain address when CapDoiGCN=false!")
				}
				if strings.Contains(partLower, `w14:checked w14:val="1"`) {
					t.Errorf("Checkbox should NOT be ticked when CapDoiGCN=false!")
				}
			}
		}
	}
}

func TestThayDoiWithCapDoiGCN(t *testing.T) {
	data := model.ThayDoiHKD{
		HoVaTen: "Lê Văn Tám",
		TenHKD:  "HỘ KINH DOANH HOA TƯƠI ĐÀ LẠT",
		MaHKD:   "0319888999",
		CoQuan:  "Phường 5",
		ThayDoiTruSo: &model.ThayDoiTruSo{
			SoNha: "456 Đường Nguyễn Huệ",
			SDT:   "0988776655",
		},
		ThayDoiNganhNghe: &model.ThayDoiNganhNghe{
			BoSung: []model.NganhNgheKinhDoanh{
				{
					TenNganh:     "Bán lẻ hoa, cây cảnh",
					MaNganh:      4773,
					ChiTiet:      "Bán hoa hồng, hoa lan",
					LaNganhChinh: true,
				},
			},
		},
		SoLaoDong: "10",
		CapDoiGCN: true,
	}

	outPath, err := FillThayDoiTemplate(data, "../templates/1.docx")
	if err != nil {
		t.Fatalf("FillThayDoiTemplate error: %v", err)
	}
	defer os.Remove(outPath)

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("zip.OpenReader error: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)
			t.Logf("Checking NgaySinh in docx: contains ngaysinh=%v, contains 15/08/1995=%v", strings.Contains(xmlStr, "ngaysinh"), strings.Contains(xmlStr, "15/08/1995"))
			idxNs := strings.Index(xmlStr, "Ngày sinh:")
			if idxNs != -1 {
				t.Logf("After Ngày sinh: in docx: %s", xmlStr[idxNs:idxNs+200])
			}
			posCapDoi := strings.Index(xmlStr, "ĐĂNG KÝ CẤP ĐỔI SANG GIẤY CHỨNG NHẬN")
			if posCapDoi == -1 {
				t.Fatalf("Document does not contain CapDoi section")
			}
			partLower := xmlStr[posCapDoi:]

			// Checkbox phải được tick
			if !strings.Contains(partLower, `w:sym w:font="Wingdings" w:char="F0FE"`) {
				t.Errorf("Wingdings checkbox should be F0FE!")
			}

			// Phần Cấp đổi phải có tên HKD và trụ sở
			if !strings.Contains(partLower, "HOA TƯƠI ĐÀ LẠT") {
				t.Errorf("Lower part must contain HKD name when CapDoiGCN=true!")
			}
			if !strings.Contains(partLower, "456 Đường Nguyễn Huệ") {
				t.Errorf("Lower part must contain address when CapDoiGCN=true!")
			}

			// Ngành nghề Table 9 phải có 4773
			if !strings.Contains(partLower, "4773") {
				t.Errorf("Lower Table 9 must contain industry code 4773!")
			}

			// Số lao động phải là 10
			if !strings.Contains(partLower, "10") {
				t.Errorf("Lower part must contain SoLaoDong 10!")
			}
		}
	}
}

func TestPlaceholderFormattingAndRules(t *testing.T) {
	// Case 1: Không có ThayDoiTen, không có ThongTinThem, không có SDT, CapDoiGCN = false
	dataEmpty := model.ThayDoiHKD{
		HoVaTen:   "Lê Thị Mỹ",
		TenHKD:    "HỘ KINH DOANH HOA LAN",
		MaHKD:     "0311223344",
		CoQuan:    "Phường 10",
		CapDoiGCN: false,
	}

	outPath, err := FillThayDoiTemplate(dataEmpty, "../templates/1.docx")
	if err != nil {
		t.Fatalf("FillThayDoiTemplate error: %v", err)
	}
	defer os.Remove(outPath)

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("zip.OpenReader error: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)

			// 1. ten_hkd_moi không được còn sót dạng {ten_hkd_moi} hay {ten_ ... moi }
			if strings.Contains(xmlStr, "ten_hkd_moi") || strings.Contains(xmlStr, "moi }") {
				t.Errorf("ten_hkd_moi was not replaced!")
			}

			// 2. nganhnghe_sua phải là rỗng ""
			if strings.Contains(xmlStr, "nganhnghe_sua") {
				t.Errorf("nganhnghe_sua was not replaced!")
			}

			// 3. thongtinthem phải là rỗng ""
			if strings.Contains(xmlStr, "thongtinthem") {
				t.Errorf("thongtinthem was not replaced!")
			}

			// 4. Điện thoại trống phải có chấm dài ~36 chấm (không phải chỉ 16 hay 20 chấm)
			if !strings.Contains(xmlStr, dotsPhone) {
				t.Errorf("Phone line dots too short, expected dotsPhone!")
			}

			// 5. Kiểm tra không còn bất kỳ placeholder dạng {abc} nào bị sót
			reUnresolved := regexp.MustCompile(`\{[a-z_0-9\s]{3,30}\}`)
			matches := reUnresolved.FindAllString(xmlStr, -1)
			if len(matches) > 0 {
				t.Errorf("Found unresolved placeholders in docx: %v", matches)
			}
		}
	}

	// Case 2: CÓ ThayDoiTen mới
	dataWithTen := model.ThayDoiHKD{
		HoVaTen: "Lê Thị Mỹ",
		TenHKD:  "HỘ KINH DOANH HOA LAN",
		MaHKD:   "0311223344",
		CoQuan:  "Phường 10",
		ThayDoiTen: &model.ThayDoiTenHKD{
			TenHKDMoi: "hộ kinh doanh hoa phong lan đà lạt",
		},
		CapDoiGCN: false,
	}

	outPath2, err := FillThayDoiTemplate(dataWithTen, "../templates/1.docx")
	if err != nil {
		t.Fatalf("FillThayDoiTemplate error: %v", err)
	}
	defer os.Remove(outPath2)

	r2, err := zip.OpenReader(outPath2)
	if err != nil {
		t.Fatalf("zip.OpenReader error: %v", err)
	}
	defer r2.Close()

	for _, f := range r2.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("f.Open error: %v", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("io.ReadAll error: %v", err)
			}

			xmlStr := string(b)
			if !strings.Contains(xmlStr, "HỘ KINH DOANH HOA PHONG LAN ĐÀ LẠT") {
				t.Errorf("New HKD name uppercase not found in output docx!")
			}
		}
	}
}
