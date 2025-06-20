package service

import (
	"baliance.com/gooxml/document"
	"fmt"
	"hkd.nam2507/model"
	"strconv"
	"strings"
	"time"
)

const HKDTemplatePath = "templates/1.docx"
const HKDOutputPath = "output"

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

func fillHKDTemplate(data model.Hokinhdoanh) error {
	doc, err := document.Open(HKDTemplatePath)
	if err != nil {
		fmt.Println("Error opening document:", err)
		return err
	}

	now := time.Now()

	placeholders := map[string]string{
		"{ngay}":           strconv.Itoa(now.Day()),
		"{thang}":          strconv.Itoa(int(now.Month())),
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
		"{quanhuyen1}":     data.DiaChiThuongTru.QuanHuyen,
		"{tinhtp1}":        data.DiaChiThuongTru.TinhTP,
		"{sonha2}":         data.DiaChiLienLac.SoNha,
		"{xaphuong2}":      data.DiaChiLienLac.XaPhuong,
		"{quanhuyen2}":     data.DiaChiLienLac.QuanHuyen,
		"{tinhtp2}":        data.DiaChiLienLac.TinhTP,
		"{sonha3}":         data.DiaChiKinhDoanh.SoNha,
		"{xaphuong3}":      data.DiaChiKinhDoanh.XaPhuong,
		"{quanhuyen3}":     data.DiaChiKinhDoanh.QuanHuyen,
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

	err = doc.SaveToFile(HKDOutputPath + "/" + data.SDT + ".docx")
	if err != nil {
		fmt.Println("Error saving document:", err)
		return err
	}

	return nil
}

func replaceInParagraph(para document.Paragraph, placeholders map[string]string) {
	var textBuilder strings.Builder
	runs := para.Runs()
	for _, run := range runs {
		textBuilder.WriteString(run.Text())
	}
	text := textBuilder.String()

	changed := false
	for placeholder, value := range placeholders {
		if strings.Contains(text, placeholder) {
			text = strings.ReplaceAll(text, placeholder, value)
			changed = true
		}
	}

	if changed {
		for _, run := range runs {
			run.ClearContent()
		}
		if len(runs) > 0 {
			runs[0].AddText(text)
		}
	}
}
func replaceInParagraph1(para document.Paragraph, placeholders map[string]string) {
	var textBuilder strings.Builder
	runs := para.Runs()
	for _, run := range runs {
		textBuilder.WriteString(run.Text())
	}

	// Ghép lại toàn bộ text, loại bỏ line breaks do Word gây ra
	text := strings.ReplaceAll(textBuilder.String(), "\n", "")
	text = strings.ReplaceAll(text, "\r", "")

	changed := false
	for placeholder, value := range placeholders {
		if strings.Contains(text, placeholder) {
			text = strings.ReplaceAll(text, placeholder, value)
			changed = true
		}
	}

	if changed {
		for _, run := range runs {
			run.ClearContent()
		}
		if len(runs) > 0 {
			runs[0].AddText(text)
		}
	}
}

func fillNganhNghePlaceholders(placeholders map[string]string, list []model.NganhNgheKinhDoanh) {
	for i := 0; i < 3; i++ {
		stt := fmt.Sprintf("{stt%d}", i+1)
		ten := fmt.Sprintf("{tennganhnghe%d}", i+1)
		ma := fmt.Sprintf("{manganh%d}", i+1)
		chinh := fmt.Sprintf("{nganhnghechinh%d}", i+1)

		if i < len(list) {
			nn := list[i]
			placeholders[stt] = strconv.Itoa(i + 1)
			placeholders[ten] = safe(nn.TenNganh)
			placeholders[ma] = strconv.Itoa(nn.MaNganh)
			placeholders[chinh] = convertBoolToString(nn.NganhNgheChinh)
		} else {
			placeholders[stt] = ""
			placeholders[ten] = ""
			placeholders[ma] = ""
			placeholders[chinh] = ""
		}
	}
}

func safe(s string) string {
	if s == "" {
		return ""
	}
	return s
}

func convertBoolToString(value bool) string {
	if value {
		return "X"
	}
	return ""
}
