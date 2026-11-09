package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hkd.nam2507/model"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const ThayDoiTemplatePath = "templates/1.docx"

func valOrDefault(val string, defaultVal string) string {
	if strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

const dotsPhone = "...................................."                           // 36 dots, cân đối "Điện thoại: " trước "Fax (nếu có):"
const dotsCoQuan = "......................................"                        // 38 dots, tiếp sau "Xã/Phường/Đặc khu: " (19 ký tự)
const dotsThanhPho = "..................."                                          // 19 dots, tiếp sau "Tỉnh/Thành phố trực thuộc trung ương: " (38 ký tự)
const dotsFullLine = "..........................................................."  // 59 dots, lấp đầy dòng độc lập với thụt lề chuẩn
const dotsShort = "...................."                                            // 20 dots
const dotsMedium = "........................................"                       // 40 dots
const dotsLong = "................................................................" // 64 dots

func formatTenNganhWithChiTiet(tenNganh, chiTiet string) string {
	cleanTen := strings.TrimSpace(tenNganh)
	cleanChiTiet := strings.TrimSpace(chiTiet)
	if cleanChiTiet != "" {
		return fmt.Sprintf("%s (Chi tiết: %s)", cleanTen, cleanChiTiet)
	}
	return cleanTen
}

func formatPhoneNumber(s string) string {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(s, "")
	if len(digits) == 10 {
		return fmt.Sprintf("%s %s %s", digits[:4], digits[4:7], digits[7:])
	}
	if len(digits) == 11 {
		return fmt.Sprintf("%s %s %s", digits[:4], digits[4:7], digits[7:])
	}
	return s
}

func findTrStart(s string, fromIdx int) int {
	sub := s[:fromIdx]
	idx1 := strings.LastIndex(sub, "<w:tr ")
	idx2 := strings.LastIndex(sub, "<w:tr>")
	if idx1 > idx2 {
		return idx1
	}
	return idx2
}

func replaceSplitPlaceholder(xmlStr, placeholderName, replacement string) string {
	// 1. Direct replace if unsplit (exact match)
	xmlStr = strings.ReplaceAll(xmlStr, "{"+placeholderName+"}", xmlEscapeDocx(replacement))
	xmlStr = strings.ReplaceAll(xmlStr, "{"+placeholderName+" }", xmlEscapeDocx(replacement))
	xmlStr = strings.ReplaceAll(xmlStr, "{ "+placeholderName+" }", xmlEscapeDocx(replacement))
	xmlStr = strings.ReplaceAll(xmlStr, "{"+strings.ToUpper(placeholderName)+"}", xmlEscapeDocx(replacement))

	// 2. Regex to match split placeholder: { ... parts of placeholderName ... }
	// Tách theo "_" để bao quát cả trường hợp Word chèn thẻ OOXML giữa các từ
	parts := strings.Split(placeholderName, "_")
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	pattern := `\{[^{}]*?` + strings.Join(parts, `[^{}]*?`) + `[^{}]*?\}`
	re := regexp.MustCompile(`(?s)` + pattern)
	xmlStr = re.ReplaceAllString(xmlStr, xmlEscapeDocx(replacement))
	return xmlStr
}

func FillThayDoiTemplate(data model.ThayDoiHKD, templatePath string) (string, error) {
	if templatePath == "" {
		templatePath = ThayDoiTemplatePath
	}
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		if templatePath == ThayDoiTemplatePath {
			templateBytes, err = os.ReadFile("../" + ThayDoiTemplatePath)
		}
		if err != nil {
			return "", fmt.Errorf("không thể đọc file template: %w", err)
		}
	}

	r, err := zip.NewReader(bytes.NewReader(templateBytes), int64(len(templateBytes)))
	if err != nil {
		return "", fmt.Errorf("không thể mở zip template: %w", err)
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	nameForFile := strings.ToUpper(strings.TrimSpace(data.HoVaTen))
	if nameForFile == "" {
		nameForFile = strings.ToUpper(strings.TrimSpace(data.TenHKD))
	}
	folderName := sanitizeFolderName(nameForFile)
	if folderName == "" {
		folderName = "ho_so_thay_doi"
	}
	outputDir := filepath.Join(getOutputPath(), folderName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("không thể tạo thư mục lưu hồ sơ: %w", err)
	}
	fileName := fmt.Sprintf("GIẤY ĐỀ NGHỊ ĐĂNG KÝ THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH_%s_0001.docx", nameForFile)
	outputFile := filepath.Join(outputDir, fileName)

	tenHKDTop := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(data.TenHKD)), "HỘ KINH DOANH ")
	tenHKDTop = strings.TrimPrefix(tenHKDTop, "HKD ")
	tenHKDTop = strings.TrimSpace(tenHKDTop)
	if tenHKDTop == "" {
		tenHKDTop = strings.ToUpper(strings.TrimSpace(data.TenHKD))
	}

	// Xác định các giá trị cơ bản
	valCoQuanTruSo := valOrDefault(data.CoQuan, dotsCoQuan)
	valThanhPho := "Thành phố Hồ Chí Minh"
	valSoNha := dotsFullLine
	valSDT := " " + dotsPhone + " "

	if data.ThayDoiTruSo != nil {
		if strings.TrimSpace(data.ThayDoiTruSo.CoQuan) != "" {
			valCoQuanTruSo = data.ThayDoiTruSo.CoQuan
		}
		if strings.TrimSpace(data.ThayDoiTruSo.ThanhPho) != "" {
			valThanhPho = data.ThayDoiTruSo.ThanhPho
		}
		if strings.TrimSpace(data.ThayDoiTruSo.SoNha) != "" {
			valSoNha = data.ThayDoiTruSo.SoNha
		}
		if strings.TrimSpace(data.ThayDoiTruSo.SDT) != "" {
			formattedPhone := formatPhoneNumber(strings.TrimSpace(data.ThayDoiTruSo.SDT))
			valSDT = " " + formattedPhone + "                  "
		}
	}

	valVonMoi := dotsMedium
	valThoiGianVon := dotsShort
	valHinhThucVon := dotsShort
	if data.ThayDoiVon != nil {
		if strings.TrimSpace(data.ThayDoiVon.VonKinhDoanhMoi) != "" {
			valVonMoi = data.ThayDoiVon.VonKinhDoanhMoi
		}
		if strings.TrimSpace(data.ThayDoiVon.ThoiGianThayDoi) != "" {
			valThoiGianVon = data.ThayDoiVon.ThoiGianThayDoi
		}
		if strings.TrimSpace(data.ThayDoiVon.HinhThucThayDoi) != "" {
			valHinhThucVon = data.ThayDoiVon.HinhThucThayDoi
		}
	}

	valTenMoi := dotsFullLine
	if data.ThayDoiTen != nil && strings.TrimSpace(data.ThayDoiTen.TenHKDMoi) != "" {
		valTenMoi = strings.ToUpper(strings.TrimSpace(data.ThayDoiTen.TenHKDMoi))
	}

	valTTT := ""
	if strings.TrimSpace(data.ThongTinThem) != "" {
		valTTT = data.ThongTinThem
	}

	var boSungList []model.NganhNgheKinhDoanh
	var boList []model.NganhNgheKinhDoanh
	if data.ThayDoiNganhNghe != nil {
		boSungList = data.ThayDoiNganhNghe.BoSung
		boList = data.ThayDoiNganhNghe.Bo

		// Quy tắc: Chỉ được chọn tối đa 1 ngành chính duy nhất trong mỗi bảng
		foundMain := false
		for i := range boSungList {
			if boSungList[i].LaNganhChinh {
				if !foundMain {
					foundMain = true
				} else {
					boSungList[i].LaNganhChinh = false
				}
			}
		}

		foundMainBo := false
		for i := range boList {
			if boList[i].LaNganhChinh {
				if !foundMainBo {
					foundMainBo = true
				} else {
					boList[i].LaNganhChinh = false
				}
			}
		}
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}

		if f.Name == "word/document.xml" {
			xmlStr := string(content)

			posCapDoi := strings.Index(xmlStr, "ĐĂNG KÝ CẤP ĐỔI SANG GIẤY CHỨNG NHẬN")
			var partUpper, partLower string
			if posCapDoi != -1 {
				partUpper = xmlStr[:posCapDoi]
				partLower = xmlStr[posCapDoi:]
			} else {
				partUpper = xmlStr
				partLower = ""
			}

			// ==========================================
			// XỬ LÝ PHẦN TRÊN (partUpper)
			// ==========================================

			// 1. Thay đổi tên hộ kinh doanh (ưu tiên thay thế trước để tránh xung đột với ten_hkd)
			partUpper = replaceSplitPlaceholder(partUpper, "ten_hkd_moi", valTenMoi)
			reTenMoi := regexp.MustCompile(`(?s)\{[^{}]*?ten_[^{}]*?moi\s*\}`)
			partUpper = reTenMoi.ReplaceAllString(partUpper, xmlEscapeDocx(valTenMoi))
			partUpper = strings.ReplaceAll(partUpper, "{ten_hkd_moi }", xmlEscapeDocx(valTenMoi))
			partUpper = strings.ReplaceAll(partUpper, "{ten_hkd_moi}", xmlEscapeDocx(valTenMoi))

			// 2. Tên HKD ở góc trên bên trái & Header
			partUpper = replaceSplitPlaceholder(partUpper, "ten_hkd", tenHKDTop)
			partUpper = strings.ReplaceAll(partUpper, "{ten_hkd}", xmlEscapeDocx(tenHKDTop))
			partUpper = strings.ReplaceAll(partUpper, "{TEN_HKD}", xmlEscapeDocx(tenHKDTop))

			// 3. Ngày tháng năm
			for ph, val := range currentDatePlaceholders() {
				partUpper = strings.ReplaceAll(partUpper, ph, xmlEscapeDocx(val))
			}

			// 4. Header hành chính
			partUpper = replaceSplitPlaceholder(partUpper, "hovaten", strings.ToUpper(valOrDefault(data.HoVaTen, dotsMedium)))
			partUpper = replaceSplitPlaceholder(partUpper, "ho_va_ten", strings.ToUpper(valOrDefault(data.HoVaTen, dotsMedium)))
			partUpper = replaceSplitPlaceholder(partUpper, "co_quan", valOrDefault(data.CoQuan, dotsMedium))
			partUpper = replaceSplitPlaceholder(partUpper, "ma_hkd", valOrDefault(data.MaHKD, dotsShort))

			// 5. Thay đổi trụ sở
			partUpper = replaceSplitPlaceholder(partUpper, "sonha", valSoNha)
			partUpper = replaceSplitPlaceholder(partUpper, "co_quan", valCoQuanTruSo)
			reThanhPhoUpper := regexp.MustCompile(`(?s)\{[^{}]*?thanh[^{}]*?pho[^{}]*?\}`)
			partUpper = reThanhPhoUpper.ReplaceAllString(partUpper, xmlEscapeDocx(valThanhPho))
			partUpper = replaceSplitPlaceholder(partUpper, "thanh_pho", valThanhPho)
			partUpper = strings.ReplaceAll(partUpper, "<w:t>{sdt}</w:t>", `<w:t xml:space="preserve">`+xmlEscapeDocx(valSDT)+`</w:t>`)
			partUpper = replaceSplitPlaceholder(partUpper, "sdt", valSDT)

			// 6. Thay đổi vốn kinh doanh
			partUpper = replaceSplitPlaceholder(partUpper, "vonkinhdoanhmoi", valVonMoi)
			partUpper = replaceSplitPlaceholder(partUpper, "vonkinhdoanh", valVonMoi)
			partUpper = replaceSplitPlaceholder(partUpper, "thoigianthaydoivon", valThoiGianVon)
			partUpper = replaceSplitPlaceholder(partUpper, "hinhthucthaydoivon", valHinhThucVon)

			// 7. Ngành nghề bổ sung (Bảng 5)
			if idxBs := strings.Index(partUpper, "stt_bosung"); idxBs != -1 {
				trStart := findTrStart(partUpper, idxBs)
				trEndRel := strings.Index(partUpper[idxBs:], "</w:tr>")
				if trStart != -1 && trEndRel != -1 {
					trEnd := idxBs + trEndRel + len("</w:tr>")
					newRows := buildNganhNgheRowsXml(boSungList, "1651", "2979", "1017", "3780", true)
					partUpper = partUpper[:trStart] + newRows + partUpper[trEnd:]
				}
			}

			// Bảng 6: Bỏ bớt
			idxBo := strings.Index(partUpper, "nganhnghe_bo<")
			if idxBo == -1 {
				idxBo = strings.Index(partUpper, "nganhnghe_bo}")
			}
			if idxBo == -1 {
				idxBo = strings.Index(partUpper, "stt_bo<")
			}
			if idxBo != -1 {
				trStart := findTrStart(partUpper, idxBo)
				trEndRel := strings.Index(partUpper[idxBo:], "</w:tr>")
				if trStart != -1 && trEndRel != -1 {
					trEnd := idxBo + trEndRel + len("</w:tr>")
					newRows := buildNganhNgheRowsXml(boList, "1651", "2979", "1017", "3780", true)
					partUpper = partUpper[:trStart] + newRows + partUpper[trEnd:]
				}
			}

			// Bảng 7 (Sửa ngành chính) & dọn dẹp các placeholder ngành nghề
			// Yêu cầu: {stt_sua}, {nganhnghe_sua}, {ma_sua} để trống "" nếu không có data
			partUpper = replaceSplitPlaceholder(partUpper, "stt_sua", "")
			partUpper = replaceSplitPlaceholder(partUpper, "ma_sua", "")
			partUpper = replaceSplitPlaceholder(partUpper, "nganhnghe_sua", "")
			partUpper = replaceSplitPlaceholder(partUpper, "stt_bosung", dotsShort)
			partUpper = replaceSplitPlaceholder(partUpper, "nganhnghe_bosung", dotsMedium)
			partUpper = replaceSplitPlaceholder(partUpper, "ma_bosung", dotsShort)
			partUpper = replaceSplitPlaceholder(partUpper, "nganh_bs_chinh", "")
			partUpper = replaceSplitPlaceholder(partUpper, "stt_bo", dotsShort)
			partUpper = replaceSplitPlaceholder(partUpper, "nganhnghe_bo", dotsMedium)
			partUpper = replaceSplitPlaceholder(partUpper, "ma_bo", dotsShort)
			partUpper = replaceSplitPlaceholder(partUpper, "nganh_bo_chinh", "")
			partUpper = replaceSplitPlaceholder(partUpper, "nganh_chinh", "")

			// 8. Đăng ký cập nhật, bổ sung, hiệu đính thông tin ({thongtinthem})
			// Yêu cầu: nếu không có data thì để trống ""
			partUpper = replaceSplitPlaceholder(partUpper, "thongtinthem", valTTT)
			partUpper = strings.ReplaceAll(partUpper, "{thongtinthem}", xmlEscapeDocx(valTTT))

			// =========================================================================
			// XỬ LÝ PHẦN DƯỚI: ĐĂNG KÝ CẤP ĐỔI SANG GIẤY CHỨNG NHẬN (partLower)
			// =========================================================================
			if partLower != "" {
				reThanhPho := regexp.MustCompile(`(?s)\{[^{}]*?thanh[^{}]*?\}`)

				if data.CapDoiGCN {
					// Tick ô Cá nhân (val="1", ký hiệu F0FE)
					partLower = strings.Replace(partLower, `w14:checked w14:val="0"`, `w14:checked w14:val="1"`, 1)
					partLower = strings.Replace(partLower, `w:sym w:font="Wingdings" w:char="F06F"`, `w:sym w:font="Wingdings" w:char="F0FE"`, 1)

					// 1. Tên HKD
					partLower = replaceSplitPlaceholder(partLower, "ten_hkd", tenHKDTop)

					// 2. Trụ sở HKD
					partLower = replaceSplitPlaceholder(partLower, "sonha", valSoNha)
					partLower = replaceSplitPlaceholder(partLower, "co_quan", valCoQuanTruSo)
					partLower = reThanhPho.ReplaceAllString(partLower, xmlEscapeDocx(valThanhPho))
					partLower = strings.ReplaceAll(partLower, "<w:t>{sdt}</w:t>", `<w:t xml:space="preserve">`+xmlEscapeDocx(valSDT)+`</w:t>`)
					partLower = replaceSplitPlaceholder(partLower, "sdt", valSDT)

					// 3. Bảng ngành nghề (Bảng 9)
					if idxT9 := strings.Index(partLower, "ten_nganh"); idxT9 != -1 {
						trStart := findTrStart(partLower, idxT9)
						trEndRel := strings.Index(partLower[idxT9:], "</w:tr>")
						if trStart != -1 && trEndRel != -1 {
							trEnd := idxT9 + trEndRel + len("</w:tr>")
							if len(boSungList) > 0 {
								partLower = partLower[:trStart] + buildNganhNgheRowsXml(boSungList, "1533", "4587", "1620", "1980", true) + partLower[trEnd:]
							} else {
								partLower = partLower[:trStart] + buildNganhNgheRowsXml(nil, "1533", "4587", "1620", "1980", true) + partLower[trEnd:]
							}
						}
					}

					// 4. Vốn kinh doanh
					partLower = replaceSplitPlaceholder(partLower, "vonkinhdoanh", valVonMoi)

					// 5.2 Số lao động (Ô số 6)
					valSLD := ""
					if strings.TrimSpace(data.SoLaoDong) != "" {
						valSLD = data.SoLaoDong
					}
					partLower = replaceSplitPlaceholder(partLower, "solaodong", valSLD)

				} else {
					// Người dùng KHÔNG chọn cấp đổi:
					// Chuyển toàn bộ các placeholder bên dưới về format .... cân đối lề phải hoặc để trống
					partLower = replaceSplitPlaceholder(partLower, "ten_hkd", dotsFullLine)
					partLower = replaceSplitPlaceholder(partLower, "sonha", dotsFullLine)
					partLower = replaceSplitPlaceholder(partLower, "co_quan", dotsCoQuan)
					partLower = reThanhPho.ReplaceAllString(partLower, dotsThanhPho)
					valEmptyPhone := " " + dotsPhone + " "
					partLower = strings.ReplaceAll(partLower, "<w:t>{sdt}</w:t>", `<w:t xml:space="preserve">`+xmlEscapeDocx(valEmptyPhone)+`</w:t>`)
					partLower = replaceSplitPlaceholder(partLower, "sdt", valEmptyPhone)

					// Bảng 9 chuyển về dòng trống có đủ 4 cột và khung bảng hoàn chỉnh
					if idxT9 := strings.Index(partLower, "ten_nganh"); idxT9 != -1 {
						trStart := findTrStart(partLower, idxT9)
						trEndRel := strings.Index(partLower[idxT9:], "</w:tr>")
						if trStart != -1 && trEndRel != -1 {
							trEnd := idxT9 + trEndRel + len("</w:tr>")
							partLower = partLower[:trStart] + buildNganhNgheRowsXml(nil, "1533", "4587", "1620", "1980", true) + partLower[trEnd:]
						}
					}

					partLower = replaceSplitPlaceholder(partLower, "vonkinhdoanh", dotsMedium)
					// Riêng {solaodong} để trống ""
					partLower = replaceSplitPlaceholder(partLower, "solaodong", "")
				}

				// Dọn dẹp bất kỳ placeholder nào còn sót lại trong partLower
				partLower = replaceSplitPlaceholder(partLower, "stt", dotsShort)
				partLower = replaceSplitPlaceholder(partLower, "ten_nganh", dotsMedium)
				partLower = replaceSplitPlaceholder(partLower, "ma_nganh", dotsShort)
				partLower = replaceSplitPlaceholder(partLower, "nganh_chinh", "")
				partLower = replaceSplitPlaceholder(partLower, "solaodong", "")
			}

			xmlStr = partUpper + partLower
			content = []byte(xmlStr)
		}

		fw, err := w.Create(f.Name)
		if err != nil {
			return "", err
		}
		if _, err := fw.Write(content); err != nil {
			return "", err
		}
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	if err := os.WriteFile(outputFile, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("không thể ghi file docx: %w", err)
	}

	return outputFile, nil
}

func xmlEscapeDocx(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func buildNganhNgheRowsXml(items []model.NganhNgheKinhDoanh, w0, w1, w2, w3 string, hasMainCol bool) string {
	borderXml := `<w:tcBorders><w:top w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:left w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:bottom w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:right w:val="single" w:sz="4" w:space="0" w:color="000000"/></w:tcBorders>`

	if len(items) == 0 {
		chinhColXml := ""
		if hasMainCol && w3 != "" {
			chinhColXml = fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t></w:t></w:r></w:p></w:tc>`, w3, borderXml)
		}
		return fmt.Sprintf(`<w:tr><w:trPr><w:trHeight w:val="512"/></w:trPr>`+
			`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t></w:t></w:r></w:p></w:tc>`+
			`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="left"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t></w:t></w:r></w:p></w:tc>`+
			`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t></w:t></w:r></w:p></w:tc>`+
			`%s</w:tr>`, w0, borderXml, w1, borderXml, w2, borderXml, chinhColXml)
	}

	var sb strings.Builder
	for i, item := range items {
		stt := strconv.Itoa(i + 1)
		tenNganhFull := formatTenNganhWithChiTiet(item.TenNganh, item.ChiTiet)
		maStr := ""
		if item.MaNganh > 0 {
			maStr = strconv.Itoa(item.MaNganh)
		}
		chinhStr := ""
		if item.LaNganhChinh {
			chinhStr = "X"
		}

		sb.WriteString(`<w:tr><w:trPr><w:trHeight w:val="512"/></w:trPr>`)
		sb.WriteString(fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`, w0, borderXml, xmlEscapeDocx(stt)))
		sb.WriteString(fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="left"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`, w1, borderXml, xmlEscapeDocx(tenNganhFull)))
		sb.WriteString(fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`, w2, borderXml, xmlEscapeDocx(maStr)))
		if hasMainCol && w3 != "" {
			sb.WriteString(fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%s" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:spacing w:before="60" w:after="60" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Times New Roman" w:eastAsia="Calibri" w:hAnsi="Times New Roman" w:cs="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`, w3, borderXml, xmlEscapeDocx(chinhStr)))
		}
		sb.WriteString(`</w:tr>`)
	}
	return sb.String()
}

