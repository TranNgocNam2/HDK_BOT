package service

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hkd.nam2507/model"
)

//go:embed web/index.html
var defaultIndexHtml []byte

// StartWebServer khởi chạy server web cho tính năng Thay đổi HKD
func StartWebServer(port string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/thaydoi", handleIndex)
	mux.HandleFunc("/api/nganh-nghe", handleNganhNghe)
	mux.HandleFunc("/api/thaydoi/submit", handleThayDoiSubmit)
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/download-zip", handleDownloadZip)
	mux.HandleFunc("/api/open-folder", handleOpenFolder)

	log.Printf("🌐 Web server đang chạy tại: http://localhost%s (hoặc http://127.0.0.1%s)", port, port)
	return http.ListenAndServe(port, mux)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/thaydoi" {
		http.NotFound(w, r)
		return
	}
	// Đọc từ disk nếu có để cập nhật nhanh không cần build lại, fallback sang embedded
	if data, err := os.ReadFile("web/index.html"); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(defaultIndexHtml)
}

func handleNganhNghe(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("nganh_nghe.json")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

type FileResponseItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type SubmitResponse struct {
	Success  bool               `json:"success"`
	Folder   string             `json:"folder,omitempty"`
	FullPath string             `json:"full_path,omitempty"`
	Files    []FileResponseItem `json:"files,omitempty"`
	Error    string             `json:"error,omitempty"`
}

func decodeBase64File(dataStr string) ([]byte, error) {
	if strings.TrimSpace(dataStr) == "" {
		return nil, nil
	}
	if idx := strings.Index(dataStr, ";base64,"); idx != -1 {
		dataStr = dataStr[idx+len(";base64,"):]
	}
	dataStr = strings.TrimSpace(dataStr)
	return base64.StdEncoding.DecodeString(dataStr)
}

func handleThayDoiSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(SubmitResponse{Success: false, Error: "Chỉ hỗ trợ phương thức POST"})
		return
	}

	var sub model.WebThayDoiSubmission
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		json.NewEncoder(w).Encode(SubmitResponse{Success: false, Error: "Dữ liệu JSON không hợp lệ: " + err.Error()})
		return
	}

	if strings.TrimSpace(sub.ThayDoi.HoVaTen) == "" || strings.TrimSpace(sub.ThayDoi.TenHKD) == "" {
		json.NewEncoder(w).Encode(SubmitResponse{Success: false, Error: "Vui lòng nhập đầy đủ Họ tên chủ hộ và Tên hộ kinh doanh"})
		return
	}

	nameForFile := strings.ToUpper(strings.TrimSpace(sub.ThayDoi.HoVaTen))
	folderName := sanitizeFolderName(nameForFile)
	outputDir := filepath.Join(getOutputPath(), folderName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		json.NewEncoder(w).Encode(SubmitResponse{Success: false, Error: "Không thể tạo thư mục lưu trữ: " + err.Error()})
		return
	}

	// 1. Tạo GIẤY ĐỀ NGHỊ ĐĂNG KÝ THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH (chỉ .docx)
	_, err := FillThayDoiTemplate(sub.ThayDoi, "")
	if err != nil {
		json.NewEncoder(w).Encode(SubmitResponse{Success: false, Error: "Lỗi tạo file thay đổi HKD: " + err.Error()})
		return
	}

	// 2. Tạo GIẤY ỦY QUYỀN (chỉ .docx) nếu có thông tin
	if sub.UyQuyen != nil && strings.TrimSpace(sub.UyQuyen.HoVaTen) != "" {
		if strings.TrimSpace(sub.UyQuyen.CoQuan) == "" {
			sub.UyQuyen.CoQuan = sub.ThayDoi.CoQuan
		}
		if _, err := FillUyQuyenTemplate(*sub.UyQuyen, ""); err != nil {
			log.Printf("Cảnh báo tạo Giấy ủy quyền: %v", err)
		}
	}

	// 3. Xử lý GIẤY CHỨNG NHẬN ĐĂNG KÝ HỘ KINH DOANH CŨ (chỉ .pdf)
	if strings.TrimSpace(sub.GCNImageBase64) != "" {
		gcnBytes, err := decodeBase64File(sub.GCNImageBase64)
		if err == nil && len(gcnBytes) > 0 {
			gcnPrefix := fmt.Sprintf("GIẤY CHỨNG NHẬN ĐĂNG KÝ HỘ KINH DOANH_%s_0001", nameForFile)
			pdfGcn := filepath.Join(outputDir, gcnPrefix+".pdf")

			if bytes.HasPrefix(gcnBytes, []byte("%PDF")) {
				_ = os.WriteFile(pdfGcn, gcnBytes, 0644)
			} else {
				_ = ConvertImageToPdf(gcnBytes, pdfGcn)
			}
		}
	}

	// 4. Xử lý BẢN SAO GIẤY TỜ PHÁP LÝ CỦA CÁ NHÂN (CCCD 2 mặt - chỉ .pdf)
	frontBytes, _ := decodeBase64File(sub.CCCDTruocBase64)
	backBytes, _ := decodeBase64File(sub.CCCDSauBase64)
	if len(frontBytes) > 0 || len(backBytes) > 0 {
		cccdPrefix := fmt.Sprintf("BẢN SAO GIẤY TỜ PHÁP LÝ CỦA CÁ NHÂN_%s_0001", nameForFile)
		pdfCccd := filepath.Join(outputDir, cccdPrefix+".pdf")
		_ = ConvertCccdImagesToPdf(frontBytes, backBytes, pdfCccd)
	}

	// Xóa các file .pdf thừa nếu có sót lại từ các bản chạy cũ của tính năng thay đổi
	oldPdfThayDoi := filepath.Join(outputDir, fmt.Sprintf("GIẤY ĐỀ NGHỊ ĐĂNG KÝ THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH_%s_0001.pdf", nameForFile))
	_ = os.Remove(oldPdfThayDoi)
	oldPdfUyQuyen := filepath.Join(outputDir, fmt.Sprintf("GIẤY ỦY QUYỀN_%s_0001.pdf", nameForFile))
	_ = os.Remove(oldPdfUyQuyen)

	// 5. Thu thập danh sách đúng 4 file hồ sơ Thay đổi HKD theo quy định
	expectedFiles := []string{
		fmt.Sprintf("GIẤY ĐỀ NGHỊ ĐĂNG KÝ THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH_%s_0001.docx", nameForFile),
		fmt.Sprintf("GIẤY ỦY QUYỀN_%s_0001.docx", nameForFile),
		fmt.Sprintf("GIẤY CHỨNG NHẬN ĐĂNG KÝ HỘ KINH DOANH_%s_0001.pdf", nameForFile),
		fmt.Sprintf("BẢN SAO GIẤY TỜ PHÁP LÝ CỦA CÁ NHÂN_%s_0001.pdf", nameForFile),
	}

	var files []FileResponseItem
	for _, fName := range expectedFiles {
		fullPath := filepath.Join(outputDir, fName)
		if _, err := os.Stat(fullPath); err == nil {
			files = append(files, FileResponseItem{
				Name: fName,
				URL:  fmt.Sprintf("/api/download?folder=%s&file=%s", url.QueryEscape(folderName), url.QueryEscape(fName)),
			})
		}
	}

	absOutputDir, _ := filepath.Abs(outputDir)
	// Tự động mở ngay thư mục đích trên Windows Explorer để người dùng sử dụng trực tiếp các file đã lưu
	_ = exec.Command("explorer.exe", absOutputDir).Start()

	json.NewEncoder(w).Encode(SubmitResponse{
		Success:  true,
		Folder:   folderName,
		FullPath: absOutputDir,
		Files:    files,
	})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	fileName := r.URL.Query().Get("file")

	cleanFolder := filepath.Clean(folder)
	cleanFile := filepath.Clean(fileName)

	targetPath := filepath.Join(getOutputPath(), cleanFolder, cleanFile)
	if _, err := os.Stat(targetPath); err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	if strings.HasSuffix(fileName, ".pdf") {
		w.Header().Set("Content-Type", "application/pdf")
	} else if strings.HasSuffix(fileName, ".docx") {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, targetPath)
}

func handleDownloadZip(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	cleanFolder := filepath.Clean(folder)
	folderPath := filepath.Join(getOutputPath(), cleanFolder)

	if _, err := os.Stat(folderPath); err != nil {
		http.NotFound(w, r)
		return
	}

	zipName := fmt.Sprintf("ho_so_%s.zip", cleanFolder)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))

	zw := zip.NewWriter(w)
	defer zw.Close()

	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(folderPath, path)
		if err != nil {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		zf, err := zw.Create(rel)
		if err != nil {
			return nil
		}
		io.Copy(zf, f)
		return nil
	})
}

func handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	cleanFolder := filepath.Clean(folder)
	folderPath := filepath.Join(getOutputPath(), cleanFolder)

	absPath, err := filepath.Abs(folderPath)
	if err == nil {
		exec.Command("explorer.exe", absPath).Start()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}
