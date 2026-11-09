package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"baliance.com/gooxml/document"
	"hkd.nam2507/model"
)

const UyQuyenTemplatePath = "templates/uyquyen_thaydoi.docx"

func FillUyQuyenTemplate(data model.UyQuyenData, templatePath string) (string, error) {
	if templatePath == "" {
		templatePath = UyQuyenTemplatePath
	}

	doc, err := document.Open(templatePath)
	if err != nil {
		if templatePath == UyQuyenTemplatePath {
			doc, err = document.Open("../" + UyQuyenTemplatePath)
		}
		if err != nil {
			return "", fmt.Errorf("không thể mở template giấy ủy quyền: %w", err)
		}
	}

	nameForFile := strings.ToUpper(strings.TrimSpace(data.HoVaTen))
	if nameForFile == "" {
		nameForFile = "NGUOI_UY_QUYEN"
	}
	folderName := sanitizeFolderName(nameForFile)
	if folderName == "" {
		folderName = "ho_so_thay_doi"
	}

	outputDir := filepath.Join(getOutputPath(), folderName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("không thể tạo thư mục lưu hồ sơ: %w", err)
	}
	fileName := fmt.Sprintf("GIẤY ỦY QUYỀN_%s_0001.docx", nameForFile)
	outputFile := filepath.Join(outputDir, fileName)

	placeholders := currentDatePlaceholders()
	coQuanCap := strings.TrimSpace(data.CoQuanCap)
	if coQuanCap == "" {
		coQuanCap = "Cục Cảnh sát QLHC về TTXH"
	}

	coQuan := strings.TrimSpace(data.CoQuan)
	if coQuan == "" {
		coQuan = "Phường Bình Tân"
	}

	placeholders["{coquan}"] = coQuan
	placeholders["{hovaten}"] = strings.ToUpper(valOrDefault(data.HoVaTen, dotsMedium))
	placeholders["{ngaysinh}"] = valOrDefault(data.NgaySinh, dotsShort)
	placeholders["{cccd}"] = valOrDefault(data.CCCD, dotsShort)
	placeholders["{ngaycap}"] = valOrDefault(data.NgayCap, dotsShort)
	placeholders["{coquancap}"] = coQuanCap
	placeholders["{diachithuongtru}"] = valOrDefault(data.DiaChiThuongTru, dotsMedium)

	applyPlaceholders(doc, placeholders)

	if err := doc.SaveToFile(outputFile); err != nil {
		return "", fmt.Errorf("không thể lưu file docx giấy ủy quyền: %w", err)
	}

	return outputFile, nil
}

