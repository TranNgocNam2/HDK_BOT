package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/color"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"hkd.nam2507/model"
)

func TestWebServerApiAndAll8Files(t *testing.T) {
	png1 := createTestPng(300, 200, color.RGBA{R: 220, G: 60, B: 60, A: 255})
	png2 := createTestPng(300, 200, color.RGBA{R: 60, G: 220, B: 60, A: 255})
	png3 := createTestPng(300, 200, color.RGBA{R: 60, G: 60, B: 220, A: 255})

	gcnB64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png1)
	cccdTruocB64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png2)
	cccdSauB64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png3)

	payload := model.WebThayDoiSubmission{
		ThayDoi: model.ThayDoiHKD{
			HoVaTen: "Nguyễn Thị Huỳnh Như",
			TenHKD:  "HỘ KINH DOANH HOA TƯƠI",
			MaHKD:   "0319888999",
			CoQuan:  "Phường 5",
			ThayDoiTen: &model.ThayDoiTenHKD{
				TenHKDMoi: "HỘ KINH DOANH HOA TƯƠI SÀI GÒN",
			},
			ThayDoiTruSo: &model.ThayDoiTruSo{
				SoNha:     "123 Đường Số 1",
				CoQuan:    "Phường 5",
				ThanhPho:  "Thành phố Hồ Chí Minh",
				SDT:       "0901234567",
			},
			ThayDoiNganhNghe: &model.ThayDoiNganhNghe{
				BoSung: []model.NganhNgheKinhDoanh{
					{
						MaNganh:      4632,
						TenNganh:     "Bán buôn thực phẩm",
						ChiTiet:      "Bán buôn thủy sản",
						LaNganhChinh: true,
					},
					{
						MaNganh:      1811,
						TenNganh:     "In ấn",
						ChiTiet:      "In chuyển nhiệt",
						LaNganhChinh: false,
					},
				},
				Bo: []model.NganhNgheKinhDoanh{
					{
						MaNganh:  1622,
						TenNganh: "Bảo dưỡng và sửa chữa mô tô, xe máy",
					},
				},
			},
			ThayDoiVon: &model.ThayDoiVon{
				VonKinhDoanhMoi: "200.000.000 VNĐ (Hai trăm triệu đồng)",
				ThoiGianThayDoi: "09/11/2026",
				HinhThucThayDoi: "Tăng vốn",
			},
			CapDoiGCN:    true,
			SoLaoDong:    "10",
			ThongTinThem: "Đăng ký cập nhật địa chỉ kinh doanh mới",
		},
		UyQuyen: &model.UyQuyenData{
			HoVaTen:         "Nguyễn Thị Huỳnh Như",
			NgaySinh:        "15/08/1995",
			CCCD:            "079195001234",
			NgayCap:         "10/10/2021",
			CoQuanCap:       "Cục Cảnh sát QLHC về TTXH",
			DiaChiThuongTru: "123 Đường Số 1, Phường 5, Quận 11, TP.HCM",
			CoQuan:          "Phường 5",
		},
		GCNImageBase64:  gcnB64,
		CCCDTruocBase64: cccdTruocB64,
		CCCDSauBase64:   cccdSauB64,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/thaydoi/submit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleThayDoiSubmit(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Submit returned status %d", resp.StatusCode)
	}

	var submitResp SubmitResponse
	if err := json.NewDecoder(w.Body).Decode(&submitResp); err != nil {
		t.Fatalf("Decode response failed: %v", err)
	}

	if !submitResp.Success {
		t.Fatalf("Submit response reported failure: %s", submitResp.Error)
	}

	t.Logf("Submit succeeded! Folder: %s, Files count: %d", submitResp.Folder, len(submitResp.Files))
	if len(submitResp.Files) != 4 {
		t.Errorf("Expected exactly 4 files, got %d", len(submitResp.Files))
	}

	// Verify all 4 files exist
	expectedFiles := []string{
		"BẢN SAO GIẤY TỜ PHÁP LÝ CỦA CÁ NHÂN_NGUYỄN THỊ HUỲNH NHƯ_0001.pdf",
		"GIẤY CHỨNG NHẬN ĐĂNG KÝ HỘ KINH DOANH_NGUYỄN THỊ HUỲNH NHƯ_0001.pdf",
		"GIẤY ĐỀ NGHỊ ĐĂNG KÝ THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH_NGUYỄN THỊ HUỲNH NHƯ_0001.docx",
		"GIẤY ỦY QUYỀN_NGUYỄN THỊ HUỲNH NHƯ_0001.docx",
	}

	folderPath := filepath.Join(getOutputPath(), submitResp.Folder)
	defer os.RemoveAll(folderPath)

	for _, ef := range expectedFiles {
		fullPath := filepath.Join(folderPath, ef)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("Expected file missing: %s", ef)
		} else {
			t.Logf("✓ Verified file exists: %s", ef)
		}
	}

	// Test Download API
	dlReq := httptest.NewRequest(http.MethodGet, "/api/download?folder="+url.QueryEscape(submitResp.Folder)+"&file="+url.QueryEscape(expectedFiles[0]), nil)
	dlW := httptest.NewRecorder()
	handleDownload(dlW, dlReq)
	if dlW.Code != http.StatusOK {
		t.Errorf("Download failed with code: %d", dlW.Code)
	}

	// Test Download Zip API
	zipReq := httptest.NewRequest(http.MethodGet, "/api/download-zip?folder="+url.QueryEscape(submitResp.Folder), nil)
	zipW := httptest.NewRecorder()
	handleDownloadZip(zipW, zipReq)
	if zipW.Code != http.StatusOK {
		t.Errorf("Download-zip failed with code: %d", zipW.Code)
	}
	if zipW.Body.Len() == 0 {
		t.Errorf("Zip file content is empty")
	} else {
		t.Logf("✓ Verified Zip download works (%d bytes)", zipW.Body.Len())
	}
}
