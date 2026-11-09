package service

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/signintech/gopdf"

	"baliance.com/gooxml/common"
	"baliance.com/gooxml/document"
	"baliance.com/gooxml/measurement"
	"baliance.com/gooxml/schema/soo/wml"
)

// ConvertDocxToPdf chuyển file DOCX sang PDF bằng Microsoft Word COM
func ConvertDocxToPdf(docxPath, pdfPath string) error {
	absDocx, err := filepath.Abs(docxPath)
	if err != nil {
		return err
	}
	absPdf, err := filepath.Abs(pdfPath)
	if err != nil {
		return err
	}

	psScript := `$ErrorActionPreference = 'Stop'
$w = New-Object -ComObject Word.Application
$w.DisplayAlerts = 0
$w.Visible = $false
try {
    $doc = $w.Documents.Open($args[0], $false, $true)
    $doc.SaveAs([ref]$args[1], [ref]17)
    $doc.Close()
} catch {
    Write-Error $_
    exit 1
} finally {
    $w.Quit()
}
`
	tmpPs1, err := os.CreateTemp("", "convert_*.ps1")
	if err != nil {
		return err
	}
	defer os.Remove(tmpPs1.Name())

	// Ghi file với BOM UTF-8 để PowerShell đọc đúng 100% tiếng Việt
	utf8Bom := []byte{0xEF, 0xBB, 0xBF}
	tmpPs1.Write(append(utf8Bom, []byte(psScript)...))
	tmpPs1.Close()

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmpPs1.Name(), absDocx, absPdf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lỗi chuyển DOCX sang PDF qua Word COM: %v (output: %s)", err, string(out))
	}

	if _, err := os.Stat(absPdf); err != nil {
		return fmt.Errorf("file PDF không được tạo sau khi chuyển đổi: %w", err)
	}

	return nil
}

// ConvertImageToPdf chuyển 1 ảnh (PNG/JPG) hoặc lưu file PDF trực tiếp thành PDF A4
func ConvertImageToPdf(inputBytes []byte, pdfPath string) error {
	if len(inputBytes) == 0 {
		return fmt.Errorf("dữ liệu ảnh/tài liệu trống")
	}

	// Nếu đã là file PDF (magic number %PDF)
	if bytes.HasPrefix(inputBytes, []byte("%PDF")) {
		return os.WriteFile(pdfPath, inputBytes, 0644)
	}

	// Tạo file ảnh tạm để gopdf đọc
	tmpImg, err := os.CreateTemp("", "hdk_img_*.png")
	if err != nil {
		return err
	}
	defer os.Remove(tmpImg.Name())
	if _, err := tmpImg.Write(inputBytes); err != nil {
		tmpImg.Close()
		return err
	}
	tmpImg.Close()

	imgConfig, _, err := image.DecodeConfig(bytes.NewReader(inputBytes))
	imgW := float64(imgConfig.Width)
	imgH := float64(imgConfig.Height)
	if err != nil || imgW == 0 || imgH == 0 {
		imgW = 800
		imgH = 600
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	pageW := gopdf.PageSizeA4.W // 595.28
	pageH := gopdf.PageSizeA4.H // 841.89

	// Canh tỉ lệ để ảnh vừa trang A4, chừa viền 36pt
	margin := 36.0
	availW := pageW - (margin * 2)
	availH := pageH - (margin * 2)

	scale := availW / imgW
	if imgH*scale > availH {
		scale = availH / imgH
	}

	drawW := imgW * scale
	drawH := imgH * scale
	posX := margin + (availW-drawW)/2
	posY := margin + (availH-drawH)/2

	rect := &gopdf.Rect{W: drawW, H: drawH}
	if err := pdf.Image(tmpImg.Name(), posX, posY, rect); err != nil {
		return fmt.Errorf("không thể chèn ảnh vào PDF: %w", err)
	}

	return pdf.WritePdf(pdfPath)
}

// ConvertCccdImagesToPdf ghép 2 ảnh CCCD (Mặt trước + Mặt sau) vào 1 trang PDF A4 chuẩn
func ConvertCccdImagesToPdf(frontBytes, backBytes []byte, pdfPath string) error {
	// Nếu người dùng gửi trực tiếp 1 file PDF chứa 2 mặt
	if bytes.HasPrefix(frontBytes, []byte("%PDF")) {
		return os.WriteFile(pdfPath, frontBytes, 0644)
	}

	if len(frontBytes) == 0 {
		return fmt.Errorf("thiếu ảnh mặt trước CCCD")
	}

	tmpFront, err := os.CreateTemp("", "hdk_cccd_front_*.png")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFront.Name())
	tmpFront.Write(frontBytes)
	tmpFront.Close()

	var tmpBackName string
	if len(backBytes) > 0 {
		tmpBack, err := os.CreateTemp("", "hdk_cccd_back_*.png")
		if err != nil {
			return err
		}
		defer os.Remove(tmpBack.Name())
		tmpBack.Write(backBytes)
		tmpBack.Close()
		tmpBackName = tmpBack.Name()
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	pageW := gopdf.PageSizeA4.W // 595.28

	// Kích thước chuẩn hiển thị thẻ CCCD trên A4: rộng ~ 420pt, cao ~ 265pt (tỉ lệ thẻ ~1.58)
	cardW := 420.0
	cardH := 265.0
	posX := (pageW - cardW) / 2

	// Mặt trước đặt ở nửa trên trang
	posY1 := 80.0
	rect1 := &gopdf.Rect{W: cardW, H: cardH}
	if err := pdf.Image(tmpFront.Name(), posX, posY1, rect1); err != nil {
		return fmt.Errorf("không thể chèn mặt trước CCCD vào PDF: %w", err)
	}

	// Mặt sau đặt ở nửa dưới trang (nếu có)
	if tmpBackName != "" {
		posY2 := 420.0
		rect2 := &gopdf.Rect{W: cardW, H: cardH}
		if err := pdf.Image(tmpBackName, posX, posY2, rect2); err != nil {
			return fmt.Errorf("không thể chèn mặt sau CCCD vào PDF: %w", err)
		}
	}

	return pdf.WritePdf(pdfPath)
}

// CreateDocxFromImages tạo file DOCX chứa các hình ảnh căn giữa (cho CCCD hoặc Giấy chứng nhận)
func CreateDocxFromImages(imageBytesList [][]byte, docxPath string) error {
	doc := document.New()
	var tempFiles []string
	defer func() {
		for _, f := range tempFiles {
			_ = os.Remove(f)
		}
	}()

	for _, imgBytes := range imageBytesList {
		if len(imgBytes) == 0 {
			continue
		}
		// Bỏ qua nếu là file PDF thuần
		if bytes.HasPrefix(imgBytes, []byte("%PDF")) {
			continue
		}

		tmpImg, err := os.CreateTemp("", "docx_img_*.png")
		if err != nil {
			return err
		}
		tmpName := tmpImg.Name()
		tempFiles = append(tempFiles, tmpName)
		tmpImg.Write(imgBytes)
		tmpImg.Close()

		img, err := common.ImageFromFile(tmpName)
		if err != nil {
			continue
		}

		imgRef, err := doc.AddImage(img)
		if err != nil {
			continue
		}

		para := doc.AddParagraph()
		para.Properties().SetAlignment(wml.ST_JcCenter)
		run := para.AddRun()
		inlineImg, err := run.AddDrawingInline(imgRef)
		if err == nil {
			cfg, _, _ := image.DecodeConfig(bytes.NewReader(imgBytes))
			w := float64(cfg.Width)
			h := float64(cfg.Height)
			if w > 0 && h > 0 {
				targetW := 5.0 * 72.0
				targetH := targetW * (h / w)
				if targetH > 3.8*72.0 {
					targetH = 3.8 * 72.0
					targetW = targetH * (w / h)
				}
				inlineImg.SetSize(measurement.Distance(targetW), measurement.Distance(targetH))
			}
		}
	}

	return doc.SaveToFile(docxPath)
}
