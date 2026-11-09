# DANH SÁCH TOÀN BỘ PLACEHOLDER TRONG FILE TEMPLATE `1.docx`

Tài liệu này liệt kê chi tiết toàn bộ các placeholder xuất hiện trong file Word mẫu `templates/1.docx` (dùng cho tính năng **Thay đổi nội dung đăng ký Hộ kinh doanh** của hệ thống).

---

## 1. Tổng quan thống kê

* **Tên file mẫu:** `templates/1.docx`
* **File XML chứa dữ liệu:** `word/document.xml`
* **Tổng số vị trí xuất hiện placeholder:** **39 vị trí**
* **Số lượng tên placeholder duy nhất:** **24 placeholder** (một số placeholder như `{ten_hkd}`, `{co_quan}`, `{sonha}`, `{thanh_pho}`, `{sdt}` xuất hiện lặp lại ở 2 khu vực: *Phần thay đổi thông tin ở trên* và *Phần kê khai cấp đổi sang GCN ở dưới*).

---

## 2. Bảng kê chi tiết 39 vị trí Placeholder theo thứ tự xuất hiện

| STT | Ký hiệu Placeholder | Phân đoạn / Mục trong văn bản | Ý nghĩa & Dữ liệu cần điền | Quy tắc xử lý & Giá trị mặc định | Ngữ cảnh trích xuất từ file Word |
| :---: | :--- | :--- | :--- | :--- | :--- |
| **1** | `{ten_hkd}` | Góc trên bên trái trang 1 | Tên hộ kinh doanh hiện tại (dưới chữ HỘ KINH DOANH) | In hoa, lược bỏ tiền tố dư thừa | `HỘ KINH DOANH {ten_hkd}` |
| **2** | `{co_quan}` | Tiêu ngữ góc phải trên | Địa danh ban hành văn bản | Tên Phường/Xã/Thị trấn (Ô số 3) | `Số: …………..{co_quan}, ngày {ngay} tháng {thang}...` |
| **3** | `{ngay}` | Tiêu ngữ góc phải trên | Ngày nộp văn bản đề nghị | Lấy ngày hiện tại (2 chữ số, vd: `09`) | `ngày {ngay} tháng {thang} năm {nam}` |
| **4** | `{thang}` | Tiêu ngữ góc phải trên | Tháng nộp văn bản đề nghị | Lấy tháng hiện tại (2 chữ số, vd: `11`) | `ngày {ngay} tháng {thang} năm {nam}` |
| **5** | `{nam}` | Tiêu ngữ góc phải trên | Năm nộp văn bản đề nghị | Lấy năm hiện tại (4 chữ số, vd: `2026`) | `ngày {ngay} tháng {thang} năm {nam}` |
| **6** | `{co_quan}` | Dòng Kính gửi | Cơ quan tiếp nhận hồ sơ | Mặc định: Ô 3 `Ủy Ban nhân dân {co_quan}` | `Kính gửi: Ủy Ban nhân dân {co_quan}` |
| **7** | `{ten_hkd}` | Dòng 1 Header hành chính | Tên hộ kinh doanh hiện tại (ghi bằng chữ in hoa) | Lấy từ Ô 2 `Tên HKD hiện tại` | `Tên hộ kinh doanh (ghi bằng chữ in hoa): HỘ KINH DOANH {ten_hkd}` |
| **8** | `{ma_hkd}` | Dòng 2 Header hành chính | Mã số hộ kinh doanh / MST | Lấy từ Ô 4 `Mã số HKD / MST` | `Mã số hộ kinh doanh/Mã số đăng ký hộ kinh doanh/Mã số thuế: {ma_hkd}` |
| **9** | `{ten_hkd_moi }` | Đăng ký thay đổi tên HKD | Tên HKD viết bằng tiếng Việt sau khi thay đổi *(Lưu ý có dấu cách cuối thẻ & ngắt run OOXML)* | **Nếu có đổi tên:** In hoa tên HKD mới.<br>**Nếu không đổi:** Điền dòng chấm đầy đủ `................` (~59 chấm) để lấp kín và cân đối dòng. | `sau khi thay đổi (ghi bằng chữ in hoa): {ten_hkd_moi }` |
| **10** | `{sonha}` | Đăng ký thay đổi trụ sở | Số nhà, phòng, ngách, hẻm, đường, phố,... | Nếu có đổi trụ sở: điền số nhà mới. Nếu không: điền dòng chấm đầy đủ `................` | `đường/phố/đại lộ, tổ/xóm/ấp/thôn: {sonha}` |
| **11** | `{co_quan}` | Đăng ký thay đổi trụ sở | Xã/Phường/Đặc khu trụ sở mới | Dùng chung với Ô 3 hoặc cơ quan trụ sở mới. Nếu không đổi: điền `................` (~38 chấm) cân đối lề phải | `Xã/Phường/Đặc khu: {co_quan}` |
| **12** | `{thanh_pho}` | Đăng ký thay đổi trụ sở | Tỉnh/Thành phố trực thuộc trung ương | **Mặc định:** `Thành phố Hồ Chí Minh`. Nếu không có: điền `................` (~19 chấm) thẳng hàng lề phải | `Tỉnh/Thành phố trực thuộc trung ương: {thanh_pho}` |
| **13** | `{sdt}` | Đăng ký thay đổi trụ sở | Điện thoại liên hệ của HKD | Nếu có đổi: điền SĐT kèm khoảng cách chuẩn.<br>**Nếu không có:** Điền dòng chấm cân đối `....................................` (36 chấm) đứng trước chữ `Fax (nếu có):`, không bị ngắn hay dính chữ | `Điện thoại:{sdt}Fax (nếu có):` |
| **14** | `{vonkinhdoanhmoi}` | Đăng ký thay đổi vốn | Vốn kinh doanh sau khi thay đổi (số, chữ, VNĐ) | Nếu có đổi vốn: điền số vốn mới. Nếu không: điền `................` | `Vốn kinh doanh sau khi thay đổi (bằng số, bằng chữ, VNĐ): {vonkinhdoanhmoi}` |
| **15** | `{thoigianthaydoivon}` | Đăng ký thay đổi vốn | Thời điểm thay đổi vốn (ngày/tháng/năm) | Nếu có đổi vốn: điền ngày. Nếu không: điền `................` | `Thời điểm thay đổi vốn: {thoigianthaydoivon}` |
| **16** | `{hinhthucthaydoivon}` | Đăng ký thay đổi vốn | Hình thức tăng, giảm vốn | `Tăng vốn` hoặc `Giảm vốn`. Nếu không: điền `................` | `Hình thức tăng, giảm vốn: {hinhthucthaydoivon}` |
| **17** | `{stt_bosung}` | Bảng 5 (Bổ sung ngành nghề) | Số thứ tự ngành bổ sung | Tạo số dòng động tương ứng số ngành bổ sung | `{stt_bosung}{nganhnghe_bosung}{ma_bosung}{nganh_bs_chinh}` |
| **18** | `{nganhnghe_bosung}` | Bảng 5 (Bổ sung ngành nghề) | Tên ngành bổ sung (kèm chi tiết nếu có) | Cỡ chữ 14pt đồng nhất, xóa sạch placeholder | `{stt_bosung}{nganhnghe_bosung}{ma_bosung}{nganh_bs_chinh}` |
| **19** | `{ma_bosung}` | Bảng 5 (Bổ sung ngành nghề) | Mã ngành cấp 4 bổ sung | Căn giữa, cỡ chữ 14pt | `{stt_bosung}{nganhnghe_bosung}{ma_bosung}{nganh_bs_chinh}` |
| **20** | `{nganh_bs_chinh}` | Bảng 5 (Bổ sung ngành nghề) | Đánh dấu ngành nghề chính | Điền `X` nếu là ngành chính (**Quy định bắt buộc:** Chỉ được chọn tối đa 1 ngành chính), để trống nếu không | `{stt_bosung}{nganhnghe_bosung}{ma_bosung}{nganh_bs_chinh}` |
| **21** | `{stt_bo}` | Bảng 6 (Bỏ bớt ngành nghề) | Số thứ tự ngành nghề bỏ bớt | Tạo số dòng động tương ứng số ngành bỏ bớt | `{stt_bo}{nganhnghe_bo}{ma_bo}{nganh_bo_chinh}` |
| **22** | `{nganhnghe_bo}` | Bảng 6 (Bỏ bớt ngành nghề) | Tên ngành nghề bị bỏ bớt | Phân biệt chính xác với `nganhnghe_bosung` | `{stt_bo}{nganhnghe_bo}{ma_bo}{nganh_bo_chinh}` |
| **23** | `{ma_bo}` | Bảng 6 (Bỏ bớt ngành nghề) | Mã ngành bị bỏ bớt | Căn giữa, cỡ chữ 14pt | `{stt_bo}{nganhnghe_bo}{ma_bo}{nganh_bo_chinh}` |
| **24** | `{nganh_bo_chinh}` | Bảng 6 (Bỏ bớt ngành nghề) | Cột ngành chính trong bảng bỏ | Đánh dấu `X` nếu ngành bị bỏ là ngành chính (chỉ chọn tối đa 1 ngành chính). Để trống `""` nếu không phải hoặc không có dữ liệu | `{stt_bo}{nganhnghe_bo}{ma_bo}{nganh_bo_chinh}` |
| **25** | `{stt_sua}` | Bảng 7 (Sửa đổi ngành chính) | Số thứ tự ngành chính sửa đổi | **Quy định bắt buộc:** Để trống hoàn toàn `""` nếu không có dữ liệu | `Mã ngành{stt_sua}{nganhnghe_sua}{ma_sua}` |
| **26** | `{nganhnghe_sua}` | Bảng 7 (Sửa đổi ngành chính) | Tên ngành chính sửa đổi | **Quy định bắt buộc:** Để trống hoàn toàn `""` nếu không có dữ liệu (không điền dấu chấm) | `Mã ngành{stt_sua}{nganhnghe_sua}{ma_sua}` |
| **27** | `{ma_sua}` | Bảng 7 (Sửa đổi ngành chính) | Mã ngành chính sửa đổi | **Quy định bắt buộc:** Để trống hoàn toàn `""` nếu không có dữ liệu | `Mã ngành{stt_sua}{nganhnghe_sua}{ma_sua}` |
| **28** | `{thongtinthem}` | Đăng ký cập nhật, bổ sung, hiệu đính | Nội dung cập nhật, bổ sung, hiệu đính | Điền nội dung hiệu đính nếu có.<br>**Nếu không có data:** **Bắt buộc để trống hoàn toàn `""`** (không điền dấu chấm) | `đăng ký hộ kinh doanh như sau:{thongtinthem}` |
| **29** | `{ten_hkd}` | Cấp đổi GCN (Mục 1) | 1. Tên HKD (chữ in hoa) | **Nếu cấp đổi:** Điền tên HKD.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `1. Tên hộ kinh doanh (ghi bằng chữ in hoa): {ten_hkd}` |
| **30** | `{sonha}` | Cấp đổi GCN (Mục 2) | 2. Trụ sở HKD (Số nhà/đường/ấp) | **Nếu cấp đổi:** Điền số nhà.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `đường/phố/đại lộ, tổ/xóm/ấp/thôn: {sonha}` |
| **31** | `{co_quan}` | Cấp đổi GCN (Mục 2) | 2. Trụ sở HKD (Xã/Phường/Đặc khu) | **Nếu cấp đổi:** Điền Phường/Xã.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `Xã/Phường/Đặc khu: {co_quan}` |
| **32** | `{thanh_pho}` | Cấp đổi GCN (Mục 2) | 2. Trụ sở HKD (Tỉnh/Thành phố) | **Nếu cấp đổi:** Điền TP.HCM.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `Tỉnh/Thành phố trực thuộc trung ương: {thanh_pho}` |
| **33** | `{sdt}` | Cấp đổi GCN (Mục 2) | 2. Trụ sở HKD (Số điện thoại) | **Nếu cấp đổi:** Điền SĐT.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `Điện thoại: {sdt}Fax (nếu có):` |
| **34** | `{stt}` | Cấp đổi GCN (Bảng 9) | Bảng ngành nghề cấp đổi - Cột STT | **Nếu cấp đổi:** Sinh danh sách hàng động.<br>**Nếu KHÔNG cấp đổi:** Dòng chấm `....` | `STT... {stt}{ten_nganh}{ma_nganh}{nganh_chinh}` |
| **35** | `{ten_nganh}` | Cấp đổi GCN (Bảng 9) | Bảng ngành nghề cấp đổi - Tên ngành | **Nếu cấp đổi:** Điền tên ngành.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `{stt}{ten_nganh}{ma_nganh}{nganh_chinh}` |
| **36** | `{ma_nganh}` | Cấp đổi GCN (Bảng 9) | Bảng ngành nghề cấp đổi - Mã ngành | **Nếu cấp đổi:** Điền mã ngành.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `{stt}{ten_nganh}{ma_nganh}{nganh_chinh}` |
| **37** | `{nganh_chinh}` | Cấp đổi GCN (Bảng 9) | Bảng ngành nghề cấp đổi - Ngành chính | Điền `X` hoặc để trống | `{stt}{ten_nganh}{ma_nganh}{nganh_chinh}` |
| **38** | `{vonkinhdoanh}` | Cấp đổi GCN (Mục 4) | 4. Vốn kinh doanh (Tổng số) | **Nếu cấp đổi:** Điền số vốn.<br>**Nếu KHÔNG cấp đổi:** Điền `................` | `Tổng số (bằng số, bằng chữ, VNĐ): {vonkinhdoanh}` |
| **39** | `{solaodong}` | Cấp đổi GCN (Mục 5.2) | 5.2 Tổng số lao động | **Quy định:** Nếu có nhập số lao động thì điền. Nếu không có data hoặc KHÔNG cấp đổi: **Bắt buộc để trống hoàn toàn** (`""`) | `5.2Tổng số lao động: {solaodong}6. Chủ thể...` |

---

## 3. Phân nhóm Placeholder theo chức năng hệ thống

### Nhóm 1: Thông tin hành chính & Định danh (Bắt buộc)
* `{ten_hkd}`: Tên hộ kinh doanh hiện tại.
* `{ma_hkd}`: Mã số hộ kinh doanh / MST.
* `{co_quan}`: Cơ quan tiếp nhận (UBND Phường/Xã) - dùng chung cho tiêu ngữ và trụ sở.
* `{ngay}`, `{thang}`, `{nam}`: Thời gian lập văn bản (tự động lấy ngày hiện tại).

### Nhóm 2: Thông tin Thay đổi (Tùy biến theo nội dung thay đổi)
* **Đổi tên HKD:** `{ten_hkd_moi }` (kèm xử lý khoảng trắng phía sau).
* **Đổi trụ sở:** `{sonha}`, `{co_quan}`, `{thanh_pho}`, `{sdt}`.
* **Đổi vốn kinh doanh:** `{vonkinhdoanhmoi}`, `{thoigianthaydoivon}`, `{hinhthucthaydoivon}`.
* **Ngành nghề Bổ sung (Bảng 5):** `{stt_bosung}`, `{nganhnghe_bosung}`, `{ma_bosung}`, `{nganh_bs_chinh}`.
* **Ngành nghề Bỏ bớt (Bảng 6):** `{stt_bo}`, `{nganhnghe_bo}`, `{ma_bo}`, `{nganh_bo_chinh}`.
* **Ngành nghề Sửa đổi (Bảng 7):** `{stt_sua}`, `{nganhnghe_sua}`, `{ma_sua}`.
* **Thông tin thêm:** `{thongtinthem}` (hỏi người dùng có muốn nhập thêm địa chỉ mới hay không).

### Nhóm 3: Thông tin Đăng ký Cấp đổi sang Giấy chứng nhận (Mục cuối)
* **Nếu người dùng CÓ chọn Cấp đổi sang GCN:**
  * Tick chọn ô `☑ Cá nhân` (`w14:checked w14:val="1"`, Wingdings `F0FE`).
  * Điền thông tin vào: `{ten_hkd}`, `{sonha}`, `{co_quan}`, `{thanh_pho}`, `{sdt}`, `{vonkinhdoanh}`.
  * Sinh các dòng ngành nghề vào Bảng 9: `{stt}`, `{ten_nganh}`, `{ma_nganh}`, `{nganh_chinh}`.
  * Điền `{solaodong}` nếu có nhập ở Ô số 6.
* **Nếu người dùng KHÔNG chọn Cấp đổi sang GCN:**
  * Giữ nguyên ô trống `☐ Cá nhân` (`val="0"`).
  * Chuyển toàn bộ các trường `{ten_hkd}`, `{sonha}`, `{co_quan}`, `{thanh_pho}`, `{sdt}`, `{vonkinhdoanh}` về định dạng dấu chấm `................`.
  * Bảng 9 hiển thị dòng dấu chấm `....`.
  * Mục 5.2 Số lao động (`{solaodong}`) **bắt buộc để trống hoàn toàn** (`""`).

---

## 4. Các quy tắc đặc thù trong lập trình xử lý

1. **Trường bắt buộc để trống `""` (không điền dấu chấm khi không có dữ liệu):**
   * `{solaodong}`: Số lao động (Ô số 6 hoặc Mục 5.2).
   * `{stt_sua}`: Số thứ tự dòng sửa đổi ngành chính (Bảng 7).
   * `{nganhnghe_sua}`: Tên ngành nghề sửa đổi (Bảng 7).
   * `{ma_sua}`: Mã ngành sửa đổi (Bảng 7).
   * `{thongtinthem}`: Thông tin đăng ký cập nhật/hiệu đính (Mục 8).
   * `{nganh_bo_chinh}`: Cột ngành chính trong Bảng 6 (Bỏ bớt).

2. **Quy chuẩn độ dài dòng chấm `....` tỉ lệ cân đối lề phải:**
   * `{sonha}` / Dòng độc lập: Điền **59 chấm** (`dotsFullLine = "..........................................................."`) lấp kín toàn bộ dòng ngang từ lề thụt đến lề phải.
   * `{co_quan}` trong trụ sở: Tiêu đề `Xã/Phường/Đặc khu: ` (19 ký tự) đi kèm **38 chấm** (`dotsCoQuan = "......................................"`).
   * `{thanh_pho}`: Tiêu đề `Tỉnh/Thành phố trực thuộc trung ương: ` (38 ký tự) đi kèm **19 chấm** (`dotsThanhPho = "..................."`).
   * `{sdt}`: Tiêu đề `Điện thoại: ` đi kèm **36 chấm** (`dotsPhone = "...................................."`) đứng trước `Fax (nếu có):`, không bị ngắn cụt hay dính chữ vào nhãn tiếp theo.
   * `{ten_hkd_moi }`: Khi không thay đổi tên, dòng dưới tiêu đề được điền dòng chấm đầy đủ **59 chấm** (`dotsFullLine`) lấp kín dòng ngang.

3. **Quy tắc ngành nghề kinh doanh:**
   * **Chỉ chọn tối đa 1 ngành nghề chính duy nhất:** Cả ở giao diện Frontend (checkbox tự động uncheck dòng khác) và Backend (lọc giữ dòng đầu tiên được chọn) đều thực thi nghiêm ngặt quy tắc này.
   * Khi thêm ngành nghề bổ sung, chi tiết ngành được ghép theo format chuẩn: `Tên ngành (Chi tiết: Diễn giải chi tiết)`.

4. **Trường có giá trị mặc định:**
   * `{thanh_pho}`: Mặc định luôn là `Thành phố Hồ Chí Minh`.
   * `{co_quan}`: Tự động dùng chung giá trị của Ô số 3 *Cơ quan tiếp nhận* cho cả phần tiêu ngữ và trụ sở.

5. **Phân tách vùng văn bản & kỹ thuật xử lý OOXML:**
   * Chia tách tài liệu tại điểm mốc `ĐĂNG KÝ CẤP ĐỔI SANG GIẤY CHỨNG NHẬN` thành **Phần Trên (Upper)** và **Phần Dưới (Lower)** để việc xử lý format dấu chấm ở phần Cấp đổi không làm ảnh hưởng đến dữ liệu đã thay đổi ở các phần phía trên.
   * Với các placeholder bị Microsoft Word ngắt thành nhiều thẻ run `<w:r>` như `{ten_hkd_moi }`, hệ thống sử dụng regex phân tách theo dấu gạch dưới `_` để đảm bảo tìm và thay thế thành công 100%, đồng thời ưu tiên thay thế `{ten_hkd_moi}` trước `{ten_hkd}` nhằm loại trừ nguy cơ xung đột chuỗi con.
