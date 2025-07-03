package service

import (
	"baliance.com/gooxml/document"
	"fmt"
	"hkd.nam2507/model"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const HKDOutputPath = "C:\\Users\\nam\\Desktop\\GPKD"

// Cải thiện hàm replace để xử lý tốt hơn các placeholder bị chia nhỏ
func replaceInParagraphImproved(para document.Paragraph, placeholders map[string]string) {
	// Lấy toàn bộ text từ tất cả runs
	var fullText strings.Builder
	runs := para.Runs()

	for _, run := range runs {
		fullText.WriteString(run.Text())
	}

	originalText := fullText.String()
	modifiedText := originalText
	hasChanges := false

	// Thực hiện replace trên toàn bộ text
	for placeholder, value := range placeholders {
		if strings.Contains(modifiedText, placeholder) {
			modifiedText = strings.ReplaceAll(modifiedText, placeholder, value)
			hasChanges = true
		}
	}

	// Nếu có thay đổi, clear tất cả runs và set text mới vào run đầu tiên
	if hasChanges {
		for _, run := range runs {
			run.ClearContent()
		}
		if len(runs) > 0 {
			runs[0].AddText(modifiedText)
		} else {
			// Nếu không có run nào, tạo run mới
			newRun := para.AddRun()
			newRun.AddText(modifiedText)
		}
	}
}

func fillHKDTemplate(data model.Hokinhdoanh, path string) (string, error) {
	doc, err := document.Open(path)
	if err != nil {
		fmt.Println("Error opening document:", err)
		return "", err
	}

	now := time.Now()

	placeholders := map[string]string{
		"{ngay}":           "0" + strconv.Itoa(now.Day()),
		"{thang}":          "0" + strconv.Itoa(int(now.Month())),
		"{nam}":            strconv.Itoa(now.Year()),
		"{coquan}":         data.CoQuan,
		"{hovaten}":        strings.ToUpper(data.HoVaTen),
		"{gioitinh}":       data.GioiTinh,
		"{ngaysinh}":       data.NgaySinh,
		"{dantoc}":         data.DanToc,
		"{mst}":            data.MST,
		"{cccd}":           data.CCCD,
		"{ngaycap}":        data.NgayCap,
		"{coquancap}":      data.CoQuanCap,
		"{sdt}":            data.SDT,
		"{tenhokinhdoanh}": strings.ToUpper(data.TenHoKinhDoanh),
		"{bangso}":         data.VonKinhDoanh.BangSo,
		"{bangchu}":        data.VonKinhDoanh.BangChu,
		"{sonha1}":         data.DiaChiThuongTru.SoNha,
		"{xaphuong1}":      data.DiaChiThuongTru.XaPhuong,
		"{tinhtp1}":        data.DiaChiThuongTru.TinhTP,
		"{sonha2}":         data.DiaChiLienLac.SoNha,
		"{xaphuong2}":      data.DiaChiLienLac.XaPhuong,
		"{tinhtp2}":        data.DiaChiLienLac.TinhTP,
		"{sonha3}":         data.DiaChiKinhDoanh.SoNha,
		"{xaphuong3}":      data.DiaChiKinhDoanh.XaPhuong,
		"{tinhtp3}":        data.DiaChiKinhDoanh.TinhTP,
	}
	fillNganhNghePlaceholders(placeholders, data.NganhNgheKinhDoanh)

	for _, table := range doc.Tables() {
		for _, row := range table.Rows() {
			for _, cell := range row.Cells() {
				for _, para := range cell.Paragraphs() {
					replaceInParagraphImproved(para, placeholders)
				}
			}
		}
	}

	for _, para := range doc.Paragraphs() {
		replaceInParagraphImproved(para, placeholders)
	}

	// Tạo thư mục theo tên
	folderName := sanitizeFolderName(data.HoVaTen)
	outputDir := filepath.Join(HKDOutputPath, folderName)

	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return "", err
	}

	// Lưu file
	outputFile := filepath.Join(outputDir, data.SDT+".docx")
	err = doc.SaveToFile(outputFile)
	if err != nil {
		fmt.Println("Error saving document:", err)
		return "", err
	}

	return outputFile, nil
}

func fillNganhNghePlaceholders(placeholders map[string]string, list []model.NganhNgheKinhDoanh) {
	for i := 0; i < 3; i++ {
		stt := fmt.Sprintf("{s%d}", i+1)
		ten := fmt.Sprintf("{tennganhnghe%d}", i+1)
		ma := fmt.Sprintf("{manganh%d}", i+1)
		if i < len(list) {
			nn := list[i]
			placeholders[stt] = strconv.Itoa(i + 1)
			placeholders[ten] = safe(nn.TenNganh)
			placeholders[ma] = strconv.Itoa(nn.MaNganh)
		} else {
			placeholders[stt] = ""
			placeholders[ten] = ""
			placeholders[ma] = ""
		}
	}
}

func safe(s string) string {
	if s == "" {
		return ""
	}
	return s
}

func sanitizeFolderName(name string) string {
	// Loại bỏ ký tự đặc biệt, dấu, khoảng trắng dư thừa
	processed := strings.ToLower(name)
	processed = strings.ReplaceAll(processed, " ", "_")
	//reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	//processed = reg.ReplaceAllString(processed, "")
	return processed
}
