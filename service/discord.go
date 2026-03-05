package service

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/signintech/gopdf"
	"hkd.nam2507/model"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const HKDTemplatePath2 = "templates/2.docx"
const PowerOfAttorneyFile = "templates/uyquyen_dktlhkd.docx"

var NganhNgheDB []model.NganhNgheKinhDoanh

func init() {
	data, err := os.ReadFile("nganh_nghe.json")
	if err != nil {
		panic("Không thể đọc file nganh_nghe.json: " + err.Error())
	}
	if err := json.Unmarshal(data, &NganhNgheDB); err != nil {
		panic("Không thể parse nganh_nghe.json: " + err.Error())
	}
}

var Token = os.Getenv("DISCORD_BOT_TOKEN")

// ==================== ENTRY POINT ====================

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}
	switch {
	case strings.HasPrefix(m.Content, "!register_hkd"):
		handleHKDRegistration(s, m)
	case strings.HasPrefix(m.Content, "!help"):
		sendHelpMessage(s, m.ChannelID)
	case strings.HasPrefix(m.Content, "!ma"):
		listNganhNghe(s, m.ChannelID)
	default:
		// Kiểm tra nếu user đang chờ gửi ảnh
		handlePendingImages(s, m)
	}
}

// ==================== HANDLERS ====================

var pendingImageSessions = map[string]model.Hokinhdoanh{}
var pendingMu sync.Mutex

func handleHKDRegistration(s *discordgo.Session, m *discordgo.MessageCreate) {
	sendErr := func(msg string) { s.ChannelMessageSend(m.ChannelID, msg) }

	fields := parseFields(strings.TrimPrefix(m.Content, "!register_hkd"))

	if errs := validateRequiredFields(fields); len(errs) > 0 {
		sendErr("❌ Lỗi thông tin:\n" + strings.Join(errs, "\n"))
		return
	}

	nganhNghe := searchNganhNgheByMaNganh(get(fields, "Ngành nghề kinh doanh"))
	if len(nganhNghe) == 0 {
		sendErr("❌ Ngành nghề kinh doanh không hợp lệ!")
		return
	}

	sample := buildHoKinhDoanhModel(fields, nganhNghe)

	if err := fillHKDTemplate(sample, HKDTemplatePath2); err != nil {
		sendErr("❌ Lỗi tạo tài liệu: " + err.Error())
		return
	}

	if err := generatePowerOfAttorney(sample, PowerOfAttorneyFile); err != nil {
		sendErr("❌ Lỗi tạo giấy ủy quyền: " + err.Error())
		return
	}

	// Lưu session chờ ảnh CCCD
	pendingMu.Lock()
	pendingImageSessions[m.Author.ID] = sample
	pendingMu.Unlock()

	var nganhNgheList strings.Builder
	for i, nn := range sample.NganhNgheKinhDoanh {
		nganhNgheList.WriteString(fmt.Sprintf("%d. **%d** - %s\n", i+1, nn.MaNganh, strings.TrimSpace(nn.TenNganh)))
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
		"✅ Tạo hồ sơ thành công cho **%s**!\n\n📋 **Ngành nghề đăng ký:**\n%s\n📎 Vui lòng gửi 2 ảnh CCCD để hoàn tất.",
		sample.HoVaTen,
		nganhNgheList.String(),
	))
}

func handlePendingImages(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Attachments) == 0 {
		return
	}

	pendingMu.Lock()
	sample, exists := pendingImageSessions[m.Author.ID]
	pendingMu.Unlock()

	if !exists {
		return
	}

	if len(m.Attachments) < 2 {
		s.ChannelMessageSend(m.ChannelID, "⚠️ Vui lòng gửi đủ 2 ảnh CCCD!")
		return
	}

	sendErr := func(msg string) { s.ChannelMessageSend(m.ChannelID, msg) }

	// Download ảnh
	var imagePaths []string
	for i, att := range m.Attachments[:2] {
		resp, err := http.Get(att.URL)
		if err != nil {
			sendErr("❌ Không thể tải ảnh: " + err.Error())
			return
		}
		defer resp.Body.Close()

		tmpFile := fmt.Sprintf("temp_img_%s_%d.jpg", m.Author.ID, i)
		f, _ := os.Create(tmpFile)
		io.Copy(f, resp.Body)
		f.Close()
		imagePaths = append(imagePaths, tmpFile)
		defer os.Remove(tmpFile)
	}

	// Lưu PDF vào cùng thư mục hồ sơ
	folderName := sanitizeFolderName(sample.HoVaTen)
	outputDir := filepath.Join(HKDOutputPath, folderName)
	pdfPath := filepath.Join(outputDir, "cccd.pdf")

	if err := imagesToPDF(imagePaths, pdfPath); err != nil {
		sendErr("❌ Lỗi tạo PDF CCCD: " + err.Error())
		return
	}

	// Xóa session và tin nhắn chứa ảnh
	pendingMu.Lock()
	delete(pendingImageSessions, m.Author.ID)
	pendingMu.Unlock()

	// Timeout 1 phút nếu không gửi ảnh
	go func(userID string) {
		time.Sleep(1 * time.Minute)
		pendingMu.Lock()
		if _, exists := pendingImageSessions[userID]; exists {
			delete(pendingImageSessions, userID)
			pendingMu.Unlock()
			s.ChannelMessageSend(m.ChannelID, "⏰ Hết thời gian chờ ảnh CCCD cho **"+sample.HoVaTen+"**! Vui lòng tạo lại hồ sơ.")
		} else {
			pendingMu.Unlock()
		}
	}(m.Author.ID)

	s.ChannelMessageDelete(m.ChannelID, m.ID)
	s.ChannelMessageSend(m.ChannelID, "✅ Đã lưu ảnh CCCD cho "+sample.HoVaTen+"!")
}

func imagesToPDF(imagePaths []string, outputPath string) error {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	pageW := 595.0
	pageH := 842.0
	maxW := pageW * 0.5
	maxH := pageH * 0.5

	for _, imgPath := range imagePaths {
		// Đọc kích thước thực của ảnh
		f, err := os.Open(imgPath)
		if err != nil {
			return err
		}
		imgConfig, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			return err
		}

		// Tính tỉ lệ để fit trong maxW x maxH mà không bị méo
		imgW := float64(imgConfig.Width)
		imgH := float64(imgConfig.Height)
		ratio := min(maxW/imgW, maxH/imgH)

		finalW := imgW * ratio
		finalH := imgH * ratio

		// Căn giữa trang
		x := (pageW - finalW) / 2
		y := (pageH - finalH) / 2

		pdf.AddPage()
		if err := pdf.Image(imgPath, x, y, &gopdf.Rect{W: finalW, H: finalH}); err != nil {
			return fmt.Errorf("lỗi thêm ảnh %s: %w", imgPath, err)
		}
	}

	return pdf.WritePdf(outputPath)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ==================== HELPERS ====================

func validateRequiredFields(fields map[string]string) []string {
	var errors []string

	required := []string{"Họ và tên", "Địa chỉ thường trú", "Ngày sinh", "Ngày cấp CCCD"}
	for _, field := range required {
		if get(fields, field) == "" {
			errors = append(errors, fmt.Sprintf("• %s không được để trống", field))
		}
	}

	if cccd := get(fields, "CCCD"); len(strings.TrimSpace(cccd)) != 12 {
		errors = append(errors, "• CCCD phải có đúng 12 chữ số")
	}

	if sdt := get(fields, "Số điện thoại"); len(strings.TrimSpace(sdt)) != 10 {
		errors = append(errors, "• Số điện thoại phải có đúng 10 chữ số")
	}

	return errors
}

func buildHoKinhDoanhModel(fields map[string]string, nganhNghe []model.NganhNgheKinhDoanh) model.Hokinhdoanh {
	fullName := get(fields, "Họ và tên")
	diaChiThuongTru := get(fields, "Địa chỉ thường trú")
	diaChiKinhDoanh := get(fields, "Địa chỉ kinh doanh")

	// Nếu thường trú không phải HCM thì liên lạc = kinh doanh
	diaChiLienLac := get(fields, "Địa chỉ liên lạc")
	if diaChiLienLac == "" {
		if parseAddress(diaChiThuongTru).TinhTP != "Thành phố Hồ Chí Minh" {
			diaChiLienLac = diaChiKinhDoanh
		} else {
			diaChiLienLac = diaChiThuongTru
		}
	}

	return model.Hokinhdoanh{
		HoVaTen:        fullName,
		GioiTinh:       get(fields, "Giới tính", "Nam"),
		NgaySinh:       get(fields, "Ngày sinh"),
		CCCD:           get(fields, "CCCD"),
		CoQuan:         get(fields, "Cơ quan", parseAddress(diaChiKinhDoanh).XaPhuong),
		CoQuanCap:      get(fields, "Nơi cấp CCCD", "Cục Trưởng Cục Cảnh Sát Quản Lý Hành Chính Về Trật Tự Xã Hội"),
		NgayCap:        get(fields, "Ngày cấp CCCD"),
		DanToc:         get(fields, "Dân tộc", "Kinh"),
		SDT:            get(fields, "Số điện thoại"),
		TenHoKinhDoanh: get(fields, "Tên hộ kinh doanh", fullName),
		VonKinhDoanh: model.VonKinhDoanh{
			BangSo:  get(fields, "Vốn kinh doanh (Bằng Số)", "30.000.000"),
			BangChu: get(fields, "Vốn kinh doanh (Bằng Chữ)", "Ba mươi"),
		},
		DiaChiThuongTru:    parseAddress(diaChiThuongTru),
		DiaChiLienLac:      parseAddress(diaChiLienLac),
		DiaChiKinhDoanh:    parseAddress(diaChiKinhDoanh),
		NganhNgheKinhDoanh: nganhNghe,
	}
}

func listNganhNghe(s *discordgo.Session, channelID string) {
	var msg strings.Builder
	msg.WriteString("**📋 Danh sách Ngành nghề Kinh doanh có sẵn:**\n\n")
	for _, nn := range NganhNgheDB {
		msg.WriteString(fmt.Sprintf("**%d** - %s\n", nn.MaNganh, nn.TenNganh))
	}
	msg.WriteString("\n*Sử dụng mã số trong lệnh đăng ký*")
	s.ChannelMessageSend(channelID, msg.String())
}

func sendHelpMessage(s *discordgo.Session, channelID string) {
	helpMsg := `**🏢 Hướng dẫn sử dụng Bot Đăng ký Hộ Kinh Doanh**

**Lệnh chính:**
` + "`!register_hkd`" + ` - Tạo giấy đề nghị đăng ký hộ kinh doanh

**Cú pháp:**
` + "```" + `
!register_hkd
Họ và tên=..................
Giới tính=Nam/Nữ
Dân tộc: Kinh
Ngày sinh=../../....
Mã số thuế=............
Địa chỉ thường trú=.............................., ...................., .......................
Ngành nghề kinh doanh=....
CCCD=............
Ngày cấp CCCD=../../....
Số điện thoại=...............
Tên hộ kinh doanh=.........................
Địa chỉ kinh doanh=........................., Phường ...................., Thành phố Hồ Chí Minh
Địa chỉ liên lạc=................................, Phường ...................., Thành phố Hồ Chí Minh
Vốn kinh Doanh (Bằng Số)=20.000.000
Vốn kinh Doanh (Bằng Chữ)=Hai mươi
` + "```" + `

**Các trường bắt buộc:** Họ và tên, CCCD (12 số), Số điện thoại (10 số), Ngày sinh, Ngày cấp CCCD, Địa chỉ thường trú, Ngành nghề kinh doanh

**Lệnh khác:**
- ` + "`!ma`" + ` - Xem danh sách mã ngành nghề
- ` + "`!help`" + ` - Hiển thị hướng dẫn này`

	s.ChannelMessageSend(channelID, helpMsg)
}

// ==================== PARSE UTILS ====================

func parseFields(raw string) map[string]string {
	res := make(map[string]string)
	for _, e := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == '\n' || r == '\r' }) {
		if kv := strings.SplitN(strings.TrimSpace(e), "=", 2); len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}
	return res
}

func get(m map[string]string, key string, defaultVal ...string) string {
	if val, ok := m[key]; ok && val != "" {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func parseAddress(full string) model.DiaChi {
	parts := strings.Split(full, ",")
	dc := model.DiaChi{}
	if len(parts) >= 1 {
		dc.SoNha = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		dc.XaPhuong = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		dc.TinhTP = strings.TrimSpace(parts[2])
	}
	return dc
}

func searchNganhNgheByMaNganh(maNganh string) []model.NganhNgheKinhDoanh {
	var result []model.NganhNgheKinhDoanh
	for _, id := range strings.Split(maNganh, ",") {
		tid := strings.TrimSpace(id)
		for _, nganh := range NganhNgheDB {
			if strings.EqualFold(nganh.TenNganh, tid) || strings.EqualFold(strconv.Itoa(nganh.MaNganh), tid) {
				result = append(result, nganh)
				break
			}
		}
	}
	return result
}
