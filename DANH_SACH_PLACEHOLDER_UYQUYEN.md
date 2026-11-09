# DANH SÁCH PLACEHOLDER TRONG FILE TEMPLATE `uyquyen_thaydoi.docx`

Tài liệu này liệt kê chi tiết toàn bộ các placeholder xuất hiện trong file Word mẫu Giấy ủy quyền `templates/uyquyen_thaydoi.docx` (dùng cho quy trình ủy quyền nộp hồ sơ thay đổi nội dung đăng ký Hộ kinh doanh).

---

## 1. Tổng quan thống kê

* **Tên file mẫu:** `templates/uyquyen_thaydoi.docx`
* **File XML chứa dữ liệu:** `word/document.xml`
* **Tổng số vị trí xuất hiện placeholder:** **15 vị trí**
* **Số lượng tên placeholder duy nhất:** **9 placeholder** (`{coquan}`, `{ngay}`, `{thang}`, `{nam}`, `{hovaten}`, `{ngaysinh}`, `{cccd}`, `{ngaycap}`, `{coquancap}`, `{diachithuongtru}`).

---

## 2. Bảng kê chi tiết 15 vị trí Placeholder

| STT | Ký hiệu Placeholder | Phân đoạn / Vị trí trong văn bản | Ý nghĩa & Dữ liệu cần điền | Quy tắc xử lý & Nguồn dữ liệu | Ngữ cảnh trích xuất từ file Word |
| :---: | :--- | :--- | :--- | :--- | :--- |
| **1** | `{coquan}` | Tiêu ngữ góc phải trên | Địa danh nơi lập giấy ủy quyền | Tên Phường/Xã/Quận/Huyện (kế thừa từ Ô 3) | `-----ooo0ooo-----{coquan}, ngày {ngay} tháng {thang} năm {nam}` |
| **2** | `{ngay}` | Tiêu ngữ góc phải trên | Ngày lập giấy ủy quyền | Tự động lấy ngày hiện tại (2 chữ số, vd: `09`) | `ngày {ngay} tháng {thang} năm {nam}` |
| **3** | `{thang}` | Tiêu ngữ góc phải trên | Tháng lập giấy ủy quyền | Tự động lấy tháng hiện tại (2 chữ số, vd: `11`) | `ngày {ngay} tháng {thang} năm {nam}` |
| **4** | `{nam}` | Tiêu ngữ góc phải trên | Năm lập giấy ủy quyền | Tự động lấy năm hiện tại (4 chữ số, vd: `2026`) | `ngày {ngay} tháng {thang} năm {nam}` |
| **5** | `{hovaten}` | Bên ủy quyền (Bên A): Ông (Bà) | Họ và tên chủ hộ kinh doanh ủy quyền | Kế thừa từ Ô 1 *Họ và tên* đã nhập ban đầu (In hoa) | `Bên ủy quyền (Bên A): Ông (Bà): {hovaten}` |
| **6** | `{ngaysinh}` | Bên ủy quyền (Bên A): Ngày sinh | Ngày tháng năm sinh của chủ hộ | Định dạng `dd/mm/yyyy` (người dùng nhập) | `Ngày sinh: {ngaysinh}` |
| **7** | `{cccd}` | Bên ủy quyền (Bên A): Số định danh | Số CCCD / Định danh cá nhân | 12 chữ số theo CCCD gắn chip (người dùng nhập) | `Số định danh cá nhân: {cccd}` |
| **8** | `{ngaycap}` | Bên ủy quyền (Bên A): Cấp ngày | Ngày cấp CCCD | Định dạng `dd/mm/yyyy` (người dùng nhập) | `Cấp ngày: {ngaycap}` |
| **9** | `{coquancap}` | Bên ủy quyền (Bên A): Nơi cấp | Cơ quan cấp CCCD | Mặc định: `Cục Cảnh sát QLHC về TTXH` | `Nơi cấp: {coquancap}` |
| **10** | `{diachithuongtru}` | Bên ủy quyền (Bên A): Thường trú | Địa chỉ nơi thường trú theo CCCD | Địa chỉ đầy đủ theo CCCD (người dùng nhập) | `Thường trú tại: {diachithuongtru}` |
| **11** | `{hovaten}` | Nội dung ủy quyền | Tên chủ hộ được ủy quyền thay mặt | Tự động điền theo `{hovaten}` ở trên | `thay mặt ông/bà {hovaten}, đến Phòng Kinh tế...` |
| **12** | `{coquan}` | Nội dung ủy quyền | Cơ quan tiếp nhận hồ sơ | Tự động điền theo `{coquan}` ở trên | `đến Phòng Kinh tế, Hạ tầng và Đô thị {coquan} để nộp hồ sơ...` |
| **13** | `{ngay}` | Thời hạn ủy quyền | Thời điểm bắt đầu ủy quyền (ngày) | Tự động lấy ngày hiện tại | `Từ ngày {ngay} tháng {thang} năm {nam}` |
| **14** | `{thang}` | Thời hạn ủy quyền | Thời điểm bắt đầu ủy quyền (tháng) | Tự động lấy tháng hiện tại | `Từ ngày {ngay} tháng {thang} năm {nam}` |
| **15** | `{nam}` | Thời hạn ủy quyền | Thời điểm bắt đầu ủy quyền (năm) | Tự động lấy năm hiện tại cho tới khi xong việc | `Từ ngày {ngay} tháng {thang} năm {nam} cho tới khi xong việc` |

---

## 3. Thông tin Bên nhận ủy quyền (Bên B) - Cố định trong template

Bên B đã được điền cố định sẵn trong template `templates/uyquyen_thaydoi.docx` như sau:
* **Ông (Bà):** TRẦN NGỌC NAM
* **Ngày sinh:** 25/07/2003
* **Số định danh cá nhân:** 075203000065
* **Cấp ngày:** 25/08/2022
* **Nơi cấp:** Cục Trưởng Cục Cảnh Sát Quản Lý Hành Chính Về Trật Tự Xã Hội
* **Thường trú tại:** 4465/9/40 Nguyễn Cửu Phú, Phường Tân Tạo, Thành phố Hồ Chí Minh
