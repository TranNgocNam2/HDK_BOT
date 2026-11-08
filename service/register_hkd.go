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

// ==================== HELPERS ====================

func sanitizeFolderName(name string) string {
	processed := strings.ToLower(name)
	processed = strings.ReplaceAll(processed, " ", "_")
	return processed
}

func safe(s string) string {
	return s
}

func currentDatePlaceholders() map[string]string {
	now := time.Now()

	date := strconv.Itoa(now.Day())
	if len(date) == 1 {
		date = "0" + date
	}

	month := strconv.Itoa(int(now.Month()))
	if len(month) == 1 {
		month = "0" + month
	}

	return map[string]string{
		"{ngay}":  date,
		"{thang}": month,
		"{nam}":   strconv.Itoa(now.Year()),
	}
}

// ==================== REPLACE LOGIC ====================

func mergeSymbolSplitRuns(para document.Paragraph) {
	runs := para.Runs()
	for i := 0; i < len(runs)-2; i++ {
		if runs[i].Text() == "{" && runs[i+2].Text() == "}" {
			merged := "{" + runs[i+1].Text() + "}"
			runs[i].ClearContent()
			runs[i].AddText(merged)
			runs[i+1].ClearContent()
			runs[i+2].ClearContent()
		}
	}
}

func replaceInParagraphSmart(para document.Paragraph, placeholders map[string]string) {
	runs := para.Runs()
	if len(runs) == 0 {
		return
	}

	// Bước 1: Replace trực tiếp từng run (placeholder không bị split)
	for _, run := range runs {
		text := run.Text()
		modified := text
		for k, v := range placeholders {
			modified = strings.ReplaceAll(modified, k, v)
		}
		if modified != text {
			run.ClearContent()
			run.AddText(modified)
		}
	}

	// Bước 2: Kiểm tra còn placeholder bị split không
	var fullText strings.Builder
	for _, run := range runs {
		fullText.WriteString(run.Text())
	}
	combined := fullText.String()

	remainingPlaceholders := map[string]string{}
	for k, v := range placeholders {
		if strings.Contains(combined, k) {
			remainingPlaceholders[k] = v
		}
	}
	if len(remainingPlaceholders) == 0 {
		return
	}

	// Bước 3: Xử lý placeholder bị split
	type runSegment struct {
		runIdx int
		start  int
		end    int
	}

	for placeholder, value := range remainingPlaceholders {
		runs = para.Runs()

		var segments []runSegment
		pos := 0
		for i, run := range runs {
			t := run.Text()
			segments = append(segments, runSegment{i, pos, pos + len(t)})
			pos += len(t)
		}

		fullStr := combined
		searchStart := 0
		for {
			idx := strings.Index(fullStr[searchStart:], placeholder)
			if idx == -1 {
				break
			}
			absStart := searchStart + idx
			absEnd := absStart + len(placeholder)

			firstRunIdx, lastRunIdx := -1, -1
			for _, seg := range segments {
				if seg.end > absStart && seg.start < absEnd {
					if firstRunIdx == -1 {
						firstRunIdx = seg.runIdx
					}
					lastRunIdx = seg.runIdx
				}
			}
			if firstRunIdx == -1 {
				break
			}

			var combinedRunText strings.Builder
			for i := firstRunIdx; i <= lastRunIdx; i++ {
				combinedRunText.WriteString(runs[i].Text())
			}
			newText := strings.ReplaceAll(combinedRunText.String(), placeholder, value)

			for i := firstRunIdx + 1; i <= lastRunIdx; i++ {
				runs[i].ClearContent()
			}
			runs[firstRunIdx].ClearContent()
			runs[firstRunIdx].AddText(newText)

			var rebuilt strings.Builder
			for _, run := range para.Runs() {
				rebuilt.WriteString(run.Text())
			}
			fullStr = rebuilt.String()
			combined = fullStr
			searchStart = absStart + len(value)
		}
	}
}

func applyPlaceholders(doc *document.Document, placeholders map[string]string) {
	for _, table := range doc.Tables() {
		for _, row := range table.Rows() {
			for _, cell := range row.Cells() {
				for _, para := range cell.Paragraphs() {
					mergeSymbolSplitRuns(para)
					replaceInParagraphSmart(para, placeholders)
				}
			}
		}
	}
	for _, para := range doc.Paragraphs() {
		mergeSymbolSplitRuns(para)
		replaceInParagraphSmart(para, placeholders)
	}
}

// ==================== GENERATE FUNCTIONS ====================

func generatePowerOfAttorney(data model.Hokinhdoanh, path string) error {
	doc, err := document.Open(path)
	if err != nil {
		return err
	}

	placeholders := currentDatePlaceholders()
	placeholders["{coquan}"] = data.CoQuan
	placeholders["{hovaten}"] = strings.ToUpper(data.HoVaTen)
	placeholders["{ngaysinh}"] = data.NgaySinh
	placeholders["{cccd}"] = data.CCCD
	placeholders["{ngaycap}"] = data.NgayCap
	placeholders["{coquancap}"] = data.CoQuanCap
	placeholders["{diachithuongtru}"] = data.DiaChiThuongTru.SoNha + ", " +
		data.DiaChiThuongTru.XaPhuong + ", " +
		data.DiaChiThuongTru.TinhTP

	applyPlaceholders(doc, placeholders)

	folderName := sanitizeFolderName(data.HoVaTen)
	outputDir := filepath.Join(HKDOutputPath, folderName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	return doc.SaveToFile(filepath.Join(outputDir, "GIẤY ỦY QUYỀN_"+strings.ToUpper(data.TenHoKinhDoanh)+"_0001.docx"))
}

func fillHKDTemplate(data model.Hokinhdoanh, path string) error {
	doc, err := document.Open(path)
	if err != nil {
		fmt.Errorf("error opening document: %w", err)
	}

	placeholders := currentDatePlaceholders()
	placeholders["{coquan}"] = data.CoQuan
	placeholders["{hovaten}"] = strings.ToUpper(data.HoVaTen)
	placeholders["{gioitinh}"] = data.GioiTinh
	placeholders["{ngaysinh}"] = data.NgaySinh
	placeholders["{dantoc}"] = data.DanToc
	placeholders["{cccd}"] = data.CCCD
	placeholders["{sdt}"] = data.SDT
	placeholders["{tenhokinhdoanh}"] = strings.ToUpper(data.TenHoKinhDoanh)
	placeholders["{bangso}"] = data.VonKinhDoanh.BangSo
	placeholders["{bangchu}"] = data.VonKinhDoanh.BangChu
	placeholders["{sonha1}"] = data.DiaChiThuongTru.SoNha
	placeholders["{xaphuong1}"] = data.DiaChiThuongTru.XaPhuong
	placeholders["{tinhtp1}"] = data.DiaChiThuongTru.TinhTP
	placeholders["{sonha2}"] = data.DiaChiLienLac.SoNha
	placeholders["{xaphuong2}"] = data.DiaChiLienLac.XaPhuong
	placeholders["{tinhtp2}"] = data.DiaChiLienLac.TinhTP
	placeholders["{sonha3}"] = data.DiaChiKinhDoanh.SoNha
	placeholders["{xaphuong3}"] = data.DiaChiKinhDoanh.XaPhuong
	placeholders["{tinhtp3}"] = data.DiaChiKinhDoanh.TinhTP
	fillNganhNghePlaceholders(placeholders, data.NganhNgheKinhDoanh)

	applyPlaceholders(doc, placeholders)

	folderName := sanitizeFolderName(data.HoVaTen)
	outputDir := filepath.Join(HKDOutputPath, folderName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Errorf("error creating directory: %w", err)
	}

	outputFile := filepath.Join(outputDir, "GIẤY ĐỀ NGHỊ ĐĂNG KÝ HỘ KINH DOANH_"+strings.ToUpper(data.HoVaTen)+"_0001"+".docx")
	if err := doc.SaveToFile(outputFile); err != nil {
		fmt.Errorf("error saving document: %w", err)
	}

	return nil
}

func fillNganhNghePlaceholders(placeholders map[string]string, list []model.NganhNgheKinhDoanh) {
	for i := 0; i < 3; i++ {
		if i < len(list) {
			placeholders[fmt.Sprintf("{s%d}", i+1)] = strconv.Itoa(i + 1)
			placeholders[fmt.Sprintf("{tennganhnghe%d}", i+1)] = safe(list[i].TenNganh)
			placeholders[fmt.Sprintf("{manganh%d}", i+1)] = strconv.Itoa(list[i].MaNganh)
		} else {
			placeholders[fmt.Sprintf("{s%d}", i+1)] = ""
			placeholders[fmt.Sprintf("{tennganhnghe%d}", i+1)] = ""
			placeholders[fmt.Sprintf("{manganh%d}", i+1)] = ""
		}
	}
}
