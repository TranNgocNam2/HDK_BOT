# HỆ THỐNG TỰ ĐỘNG HÓA HỒ SƠ ĐĂNG KÝ HỘ KINH DOANH (HDK_BOT)

Chào mừng bạn đến với dự án **HDK_BOT**! Đây là giải pháp tự động hóa toàn diện quy trình lập và xử lý hồ sơ hành chính dành cho Hộ Kinh Doanh tại Việt Nam.

---

## 📌 TÀI LIỆU DỰ ÁN CHI TIẾT
Dự án được tài liệu hóa chi tiết qua các văn bản chuyên sâu sau:
1. 📖 [**Tài liệu Đặc tả Nghiệp vụ & Kiến trúc Kỹ thuật Toàn diện**](file:///d:/Code/HDK_BOT/YEU_CAU_THAY_DOI_HKD.md) (`YEU_CAU_THAY_DOI_HKD.md`): Tài liệu trung tâm giải thích toàn bộ bối cảnh bài toán, quy trình 4 bước, bảng danh mục 4 file đầu ra, các giải pháp kỹ thuật cốt lõi và hướng dẫn cài đặt.
2. 📋 [**Danh sách Toàn bộ Placeholder trong File 1.docx**](file:///d:/Code/HDK_BOT/DANH_SACH_PLACEHOLDER_1_DOCX.md) (`DANH_SACH_PLACEHOLDER_1_DOCX.md`): Thống kê chi tiết 39 vị trí xuất hiện của 24 placeholder trong file Word Giấy đề nghị thay đổi.
3. 📋 [**Danh sách Toàn bộ Placeholder trong File Giấy Ủy Quyền**](file:///d:/Code/HDK_BOT/DANH_SACH_PLACEHOLDER_UYQUYEN.md) (`DANH_SACH_PLACEHOLDER_UYQUYEN.md`): Thống kê chi tiết 15 vị trí placeholder trong file `uyquyen_thaydoi.docx`.

---

## 🚀 TỔNG QUAN KIẾN TRÚC
Hệ thống gồm **2 phân hệ hoạt động song song**:
- **Phân hệ 1 - Đăng ký thành lập mới:** Tương tác qua **Discord Bot** (`register_hkd.go`, `discord.go`), xuất file theo biểu mẫu `templates/2.docx`.
- **Phân hệ 2 - Đăng ký thay đổi nội dung:** Chạy trên **Web Application** (`http://localhost:8080`), hỗ trợ dán ảnh từ Clipboard (`Ctrl + V`), tự động xuất trọn bộ **4 file chuẩn hóa** (2 file Word & 2 file PDF) lưu trực tiếp tại thư mục `Desktop/GPKD/{tên_chủ_hộ}/`.

---

## ⚡ HƯỚNG DẪN KHỞI CHẠY NHANH
```powershell
# Chạy file thực thi
./hkd_bot.exe

# Hoặc chạy từ mã nguồn
go run main.go
```
Truy cập giao diện Web App tại:
```text
http://localhost:8080
```

---

## 🧪 KIỂM THỬ HỆ THỐNG
```powershell
go test ./service -v -count=1
```
*Tất cả bài kiểm thử (Word COM, PDF Converter, API xuất 4 file) đều đạt kết quả `PASS` 100%.*
