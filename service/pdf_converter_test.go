package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"hkd.nam2507/model"
)

func createTestPng(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	buf := new(bytes.Buffer)
	png.Encode(buf, img)
	return buf.Bytes()
}

func TestStepByStepThayDoi(t *testing.T) {
	data := model.ThayDoiHKD{
		HoVaTen: "Nguyễn Thị Huỳnh Như",
		TenHKD:  "HỘ KINH DOANH HOA TƯƠI",
		MaHKD:   "0319888999",
		CoQuan:  "Phường 5",
	}

	testWithXml := func(stepName string, xmlStr string) bool {
		b, _ := os.ReadFile("../templates/1.docx")
		r, _ := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)
		for _, f := range r.File {
			fw, _ := w.Create(f.Name)
			if f.Name == "word/document.xml" {
				fw.Write([]byte(xmlStr))
			} else {
				rc, _ := f.Open()
				io.Copy(fw, rc)
				rc.Close()
			}
		}
		w.Close()

		tmpDocx := filepath.Join(os.TempDir(), stepName+".docx")
		os.WriteFile(tmpDocx, buf.Bytes(), 0644)
		defer os.Remove(tmpDocx)

		cmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf(`
$w = New-Object -ComObject Word.Application
$w.DisplayAlerts = 0
try {
    $doc = $w.Documents.Open('%s')
    $doc.Close()
    Write-Host 'OK'
} catch {
    Write-Host 'ERR'
} finally {
    $w.Quit()
}
`, tmpDocx))
		out, _ := cmd.CombinedOutput()
		res := strings.TrimSpace(string(out))
		t.Logf("Step [%s]: %s", stepName, res)
		return res == "OK"
	}

	b, _ := os.ReadFile("../templates/1.docx")
	r, _ := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	var origXml string
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			origXml = string(content)
			break
		}
	}

	// 1. Tên HKD
	x := replaceSplitPlaceholder(origXml, "ten_hkd", "HOA TƯƠI")
	testWithXml("1_ten_hkd", x)

	// 2. CoQuan
	x = replaceSplitPlaceholder(x, "co_quan", "Phường 5")
	testWithXml("2_co_quan", x)

	// 3. MaHKD
	x = replaceSplitPlaceholder(x, "ma_hkd", "0319888999")
	testWithXml("3_ma_hkd", x)

	// 4. SonHa
	x = replaceSplitPlaceholder(x, "sonha", "123 Đường Số 1")
	testWithXml("4_sonha", x)

	// 5. ThanhPho
	x = replaceSplitPlaceholder(x, "thanh_pho", "Thành phố Hồ Chí Minh")
	testWithXml("5_thanh_pho", x)

	// 6. SDT
	x = replaceSplitPlaceholder(x, "sdt", "0901234567")
	testWithXml("6_sdt", x)

	// 7. Von
	x = replaceSplitPlaceholder(x, "vonkinhdoanhmoi", "100.000.000 VNĐ")
	testWithXml("7_von", x)

	// 8. ThongTinThem
	x = replaceSplitPlaceholder(x, "thongtinthem", "Tòa nhà Bitexco")
	testWithXml("8_thongtinthem", x)

	// 9. Full FillThayDoiTemplate
	outPath, _ := FillThayDoiTemplate(data, "../templates/1.docx")
	defer os.Remove(outPath)
	bFull, _ := os.ReadFile(outPath)
	rFull, _ := zip.NewReader(bytes.NewReader(bFull), int64(len(bFull)))
	for _, f := range rFull.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			c, _ := io.ReadAll(rc)
			rc.Close()
			testWithXml("9_full_output", string(c))
		}
	}
}

func TestSafePlaceholderReplacement(t *testing.T) {
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
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			s := string(b)
			if !strings.Contains(s, "TRẦN NGỌC NAM") {
				t.Errorf("FATAL: TRẦN NGỌC NAM was corrupted/deleted from docx!")
			} else {
				t.Logf("TRẦN NGỌC NAM is preserved correctly!")
			}

			// Thử mở bằng Word COM
			pdfPath := strings.TrimSuffix(outPath, ".docx") + ".pdf"
			defer os.Remove(pdfPath)
			errConv := ConvertDocxToPdf(outPath, pdfPath)
			t.Logf("ConvertDocxToPdf result: %v", errConv)
			if errConv == nil {
				t.Logf("Successfully converted to PDF: %s", pdfPath)
			}
		}
	}
}

func TestCreateDocxFromImages(t *testing.T) {
	png1 := createTestPng(300, 200, color.RGBA{R: 200, G: 50, B: 50, A: 255})
	png2 := createTestPng(300, 200, color.RGBA{R: 50, G: 200, B: 50, A: 255})

	tmpDocx := filepath.Join(os.TempDir(), "test_images.docx")
	defer os.Remove(tmpDocx)

	err := CreateDocxFromImages([][]byte{png1, png2}, tmpDocx)
	if err != nil {
		t.Fatalf("CreateDocxFromImages failed: %v", err)
	}

	tmpPdf := filepath.Join(os.TempDir(), "test_images.pdf")
	defer os.Remove(tmpPdf)

	err = ConvertDocxToPdf(tmpDocx, tmpPdf)
	if err != nil {
		t.Fatalf("ConvertDocxToPdf on image docx failed: %v", err)
	}
	t.Logf("TestCreateDocxFromImages passed! Docx and PDF created successfully.")
}
